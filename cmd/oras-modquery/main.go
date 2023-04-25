package main

import (
	"context"
	"flag"
	"fmt"
	"os"
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

// TODO get module list
// TODO get module latest
