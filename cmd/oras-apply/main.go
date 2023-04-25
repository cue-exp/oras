package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"oras.land/oras-go/v2/registry/remote"

	"github.com/cue-exp/oras/orasflow"
)

const (
	debug          = true
	singleThreaded = true
)

const orasPkg = "github.com/cue-exp/oras"

var orasField = cue.MakePath(cue.Hid("_oras", orasPkg))

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: oras-apply [pkg]\n")
		os.Exit(2)
	}
	flag.Parse()

	pkg := "."
	switch flag.NArg() {
	case 0:
	case 1:
		pkg = flag.Arg(0)
	default:
		flag.Usage()
	}
	if err := runApply(pkg); err != nil {
		fmt.Fprintf(os.Stderr, "oras-apply: %v\n", err)
		os.Exit(1)
	}
}

func runApply(pkg string) error {
	inst := load.Instances([]string{pkg}, nil)[0]
	if err := inst.Err; err != nil {
		return fmt.Errorf("cannot load instance %q: %v", pkg, errors.Details(err, nil))
	}
	cueCtx := cuecontext.New()
	v := cueCtx.BuildInstance(inst)
	if err := v.Err(); err != nil {
		return fmt.Errorf("cannot build instance: %v", errors.Details(err, nil))
	}
	reg := os.Getenv("OCI_REGISTRY")
	if reg == "" {
		reg = "localhost:5000" // TODO lose this default
	}
	registry, err := remote.NewRegistry(reg)
	if err != nil {
		return fmt.Errorf("cannot make registry instance: %v", err)
	}
	registry.PlainHTTP = true
	ctx := context.Background()
	if err := registry.Ping(ctx); err != nil {
		return fmt.Errorf("cannot ping registry: %v", err)
	}
	return orasflow.Apply(ctx, v, orasflow.RegistryFromRemote(registry))
}
