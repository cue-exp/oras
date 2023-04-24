package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/tools/flow"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/tools/txtar"
	"oras.land/oras-go/v2/registry/remote"
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
	a := &applier{
		cueCtx:   cueCtx,
		registry: registry,
	}

	ctl := flow.New(&flow.Config{
		FindHiddenTasks: true,
	}, v, a.getTask)
	if err := ctl.Run(ctx); err != nil {
		return fmt.Errorf("error running flow: %v", err)
	}
	return nil
}

type applier struct {
	cueCtx   *cue.Context
	registry *remote.Registry
}

func (a *applier) getTask(v cue.Value) (flow.Runner, error) {
	otype := v.LookupPath(orasField)
	if otype.Err() != nil {
		return nil, nil
	}
	s, err := otype.String()
	if err != nil {
		return nil, fmt.Errorf("_oras field is not string")
	}
	switch s {
	case "blob":
		return flow.RunnerFunc(a.pushBlob), nil
	case "tag":
		return flow.RunnerFunc(a.pushTag), nil
	case "manifest":
		return flow.RunnerFunc(a.pushManifest), nil
	default:
		return nil, fmt.Errorf("unknown _oras field value %q", s)
	}
}

type blobPush struct {
	Desc   ocispec.Descriptor `json:"desc"`
	Repo   string             `json:"repo,omitempty"`
	Source any                `json:"source"`
}

func (a *applier) pushBlob(t *flow.Task) error {
	ctx := t.Context()
	var p blobPush
	if err := t.Value().Decode(&p); err != nil {
		return fmt.Errorf("cannot decode #blob from path %v (%v): %v", t.Path(), t.Value(), err)
	}
	repo, err := a.registry.Repository(ctx, p.Repo)
	if err != nil {
		return fmt.Errorf("cannot make repository from %q: %v", p.Repo, err)
	}
	var sourceData []byte
	switch mtype := p.Desc.MediaType; {
	case strings.HasSuffix(mtype, "json"): // TODO better
		data, err := json.Marshal(p.Source)
		if err != nil {
			return fmt.Errorf("cannot marshal JSON: %v", err)
		}
		sourceData = data
	case mtype == "text/plain":
		s, ok := p.Source.(string)
		if !ok {
			return fmt.Errorf("invalid source %#v for text/plain media type", p.Source)
		}
		sourceData = []byte(s)
	case mtype == "application/zip":
		s, ok := p.Source.(string)
		if !ok {
			return fmt.Errorf("invalid source %#v for application/zip media type", p.Source)
		}
		data, err := getZip(s)
		if err != nil {
			return fmt.Errorf("cannot make zip: %v", err)
		}
		sourceData = data
	}
	p.Desc.Digest = digest.FromBytes(sourceData)
	p.Desc.Size = int64(len(sourceData))
	log.Printf("pushing blob to %s; digest %s", p.Repo, p.Desc.Digest)
	if err := repo.Push(ctx, p.Desc, bytes.NewReader(sourceData)); err != nil {
		return fmt.Errorf("error pushing blob to repo %q: %v", p.Repo, err)
	}
	fillTaskPath(t, cue.MakePath(cue.Str("desc"), cue.Str("digest")), p.Desc.Digest)
	return nil
}

type tagPush struct {
	Repo   string `json:"repo,omitempty"`
	Name   string `json:"name"`
	Digest string `json:"digest"`
}

func (a *applier) pushTag(t *flow.Task) error {
	ctx := t.Context()
	var p tagPush
	if err := t.Value().Decode(&p); err != nil {
		return fmt.Errorf("cannot decode #blob from path %v (%v): %v", t.Path(), t.Value(), err)
	}
	repo, err := a.registry.Repository(ctx, p.Repo)
	if err != nil {
		return fmt.Errorf("cannot make repository from %q: %v", p.Repo, err)
	}
	descriptor, err := repo.Resolve(ctx, p.Digest)
	if err != nil {
		return fmt.Errorf("cannot resolve digest %q: %v", p.Digest, err)
	}
	log.Printf("pushing tag %s:%s", p.Repo, p.Name)
	if err := repo.Tag(ctx, descriptor, p.Name); err != nil {
		return fmt.Errorf("cannot create tag %q: %v", p.Name, err)
	}
	return nil
}

type manifestPush struct {
	Repo     string             `json:"repo,omitempty"`
	Manifest json.RawMessage    `json:"manifest"`
	Desc     ocispec.Descriptor `json:"desc"`
}

func (a *applier) pushManifest(t *flow.Task) error {
	var p manifestPush
	if err := t.Value().Decode(&p); err != nil {
		return fmt.Errorf("cannot decode #blob from path %v (%v): %v", t.Path(), t.Value(), err)
	}
	ctx := t.Context()
	repo, err := a.registry.Repository(ctx, p.Repo)
	if err != nil {
		return fmt.Errorf("cannot make repository from %q: %v", p.Repo, err)
	}
	manifestRepo := repo.Manifests()

	p.Desc.MediaType = ocispec.MediaTypeImageManifest
	p.Desc.Digest = digest.FromBytes(p.Manifest)
	p.Desc.Size = int64(len(p.Manifest))

	// Ensure that the generated manifest is valid
	log.Printf("pushing manifest to %s; digest %s", p.Repo, p.Desc.Digest)
	if err := manifestRepo.Push(ctx, p.Desc, bytes.NewReader(p.Manifest)); err != nil {
		return fmt.Errorf("error pushing manifest to repo %q: %v", p.Repo, err)
	}
	fillTaskPath(t, cue.MakePath(cue.Str("desc")), p.Desc)
	return nil
}

func fillTaskPath(t *flow.Task, path cue.Path, v any) {
	top := t.Value().Context().CompileString(`_`)
	t.Fill(top.FillPath(path, v))
}

// getZip returns a zip archive consisting of all the files in ar
func getZip(s string) ([]byte, error) {
	ar := txtar.Parse([]byte(s))
	var buf bytes.Buffer
	zipw := zip.NewWriter(&buf)
	nwritten := 0
	for _, f := range ar.Files {
		w, err := zipw.Create(f.Name)
		if err != nil {
			return nil, err
		}
		_, err = w.Write(f.Data)
		if err != nil {
			return nil, err
		}
		nwritten++
	}
	if err := zipw.Close(); err != nil {
		return nil, err
	}

	if nwritten == 0 {
		return nil, fmt.Errorf("no files found in txtar archive")
	}
	return buf.Bytes(), nil
}

func getFile(ar *txtar.Archive, name string) ([]byte, error) {
	name = path.Clean(name)
	for _, f := range ar.Files {
		if path.Clean(f.Name) == name {
			return f.Data, nil
		}
	}
	return nil, fmt.Errorf("file %q not found in txtar archive", name)
}
