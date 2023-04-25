package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"

	"github.com/cue-exp/oras/registryclient"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `
usage: oras-modquery [cmd [arg...]]

Sub-commands:

	modfile $module@$version
	list $module
	latest $module
	vendor $module
	deps $module@$version
`)
		os.Exit(2)
	}
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
	}
	if err := runCommand(flag.Arg(0), flag.Args()[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "oras-modquery: %v\n", err)
		os.Exit(1)
	}
}

func runCommand(cmd string, args []string) error {
	reg := os.Getenv("OCI_REGISTRY")
	if reg == "" {
		reg = "localhost:5000" // TODO lose this default
	}
	client, err := registryclient.New(reg)
	if err != nil {
		return fmt.Errorf("cannot make registry instance: %v", err)
	}
	ctx := context.Background()
	switch cmd {
	case "modfile":
		return showModFile(ctx, client, args)
	case "list":
		return listVersions(ctx, client, args)
	case "deps":
		return showDeps(ctx, client, args)
	case "latest":
		return latestVersion(ctx, client, args)
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func showModFile(ctx context.Context, client *registryclient.Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: modfile $module@$version")
	}
	mod := args[0]
	mv, ok := splitPathVersion(mod)
	if !ok {
		return fmt.Errorf("invalid module path@version %q", mod)
	}
	m, err := client.GetModule(ctx, mv)
	if err != nil {
		return err
	}
	data, err := m.ModuleFile(ctx)
	if err != nil {
		return fmt.Errorf("cannot get module file: %v", err)
	}
	os.Stdout.Write(data)
	if len(data) > 0 && data[len(data)-1] != '\n' {
		fmt.Println()
	}
	return nil
}

func showDeps(ctx context.Context, client *registryclient.Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: deps $module@$version")
	}
	mod := args[0]
	mv, ok := splitPathVersion(mod)
	if !ok {
		return fmt.Errorf("invalid module path@version %q", mod)
	}
	m, err := client.GetModule(ctx, mv)
	if err != nil {
		return err
	}
	deps, err := m.Dependencies(ctx)
	if err != nil {
		return fmt.Errorf("cannot access dependencies: %v", err)
	}
	depStrs := make([]string, 0, len(deps))
	for _, dep := range deps {
		depStrs = append(depStrs, dep.Version().String())
	}
	sort.Strings(depStrs)
	for _, s := range depStrs {
		fmt.Println(s)
	}
	return nil
}

func listVersions(ctx context.Context, client *registryclient.Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: list $module")
	}
	mod := args[0]
	if strings.Contains(mod, "@") {
		return fmt.Errorf("list does not take an @$version suffix")
	}
	versions, err := client.ModuleVersions(ctx, mod)
	if err != nil {
		return err
	}
	sort.Sort(semver.ByVersion(versions))
	for _, v := range versions {
		fmt.Println(v)
	}
	return nil
}

func latestVersion(ctx context.Context, client *registryclient.Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: latest $module")
	}
	mod := args[0]
	if strings.Contains(mod, "@") {
		return fmt.Errorf("list does not take an @$version suffix")
	}
	versions, err := client.ModuleVersions(ctx, mod)
	if err != nil {
		return err
	}
	maxPre := ""
	maxStable := ""
	for _, v := range versions {
		pre := semver.Prerelease(v)
		if pre == "" {
			if maxStable == "" || semver.Compare(v, maxStable) > 0 {
				maxStable = v
			}
		} else {
			if maxPre == "" || semver.Compare(v, maxPre) > 0 {
				maxPre = v
			}
		}
	}
	if maxStable == "" && maxPre == "" {
		// TODO log this, as it means that some of the versions aren't valid?
		return fmt.Errorf("no versions found for %q", mod)
	}
	result := maxStable
	if result == "" {
		// No stable versions; fall back to using the latest prerelease version.
		result = maxPre
	}
	if maxStable != "" {
		result = maxStable
	}
	fmt.Println(result)
	return nil
}

func splitPathVersion(m string) (module.Version, bool) {
	parts := strings.Split(m, "@")
	if len(parts) != 2 {
		return module.Version{}, false
	}
	v := module.Version{
		Path:    parts[0],
		Version: parts[1],
	}
	if !semver.IsValid(v.Version) {
		return module.Version{}, false
	}
	if semver.Canonical(v.Version) != v.Version {
		return module.Version{}, false
	}
	return v, true
}

func vendor(mod string) error {
	panic("unimplemented")
	// get manifest
	// get config
	// for each layer {
	//	if layer 0 {
	//		modname = self
	//	}
	// 	writedir(modname, blob)
	// }
}
