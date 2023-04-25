package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/tools/flow"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote"
)

const (
	debug          = true
	singleThreaded = false
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

	ctl := flow.New(nil, v, a.getTask)
	if err := ctl.Run(ctx); err != nil {
		return fmt.Errorf("error running flow: %v", err)
	}
	return nil
}

type applier struct {
	cueCtx   *cue.Context
	registry *remote.Registry
}

func logf(f string, a ...any) {
	if debug {
		fmt.Println(fmt.Sprintf(f, a...))
	}
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
		return flow.RunnerFunc(mutex(a.pushBlob)), nil
	case "tag":
		return flow.RunnerFunc(mutex(a.pushTag)), nil
	case "manifest":
		return flow.RunnerFunc(mutex(a.pushManifest)), nil
	default:
		return nil, fmt.Errorf("unknown _oras field value %q", s)
	}
}

type blobPush struct {
	Desc   ocispec.Descriptor `json:"desc"`
	Repo   string             `json:"repo,omitempty"`
	Source json.RawMessage    `json:"source"`
}

var globalMutex sync.Mutex

func mutex(f flow.RunnerFunc) flow.RunnerFunc {
	if !singleThreaded {
		return f
	}
	return func(t *flow.Task) error {
		globalMutex.Lock()
		logf("-- run %v {", t.Path())
		defer logf("}")
		defer globalMutex.Unlock()
		return f(t)
	}
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
	case strings.HasSuffix(mtype, "+json") || strings.HasSuffix(mtype, "/json"):
		sourceData = p.Source
	case mtype == "text/plain":
		var s string
		if err := json.Unmarshal(p.Source, &s); err != nil {
			return fmt.Errorf("invalid source %#v for text/plain media type", p.Source)
		}
		sourceData = []byte(s)
	case mtype == "application/zip":
		var files map[string]string
		if err := json.Unmarshal(p.Source, &files); err != nil {
			return fmt.Errorf("invalid source %#v for application/zip media type", p.Source)
		}
		data, err := getZip(files)
		if err != nil {
			return fmt.Errorf("cannot make zip: %v", err)
		}
		sourceData = data
	}
	p.Desc.Digest = digest.FromBytes(sourceData)
	p.Desc.Size = int64(len(sourceData))
	prettySource, _ := json.MarshalIndent(p.Source, "\t", "\t")
	logf("%v: push %s to %s -> %s\n\tsource: %s", t.Path(), p.Desc.MediaType, p.Repo, p.Desc.Digest, prettySource)
	if err := repo.Push(ctx, p.Desc, bytes.NewReader(sourceData)); err != nil {
		return fmt.Errorf("error pushing blob to repo %q: %v", p.Repo, err)
	}
	t.Fill(t.Value().FillPath(cue.MakePath(cue.Str("desc"), cue.Str("digest")), p.Desc.Digest))
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
	logf("%v: push tag %s:%s", t.Path(), p.Repo, p.Name)
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
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v: push manifest to %s -> %s\n", t.Path(), p.Repo, p.Desc.Digest)

	fmt.Fprintf(&buf, "\tsource: ")
	json.Indent(&buf, p.Manifest, "\t", "\t")
	logf("%s", buf.String())
	if err := manifestRepo.Push(ctx, p.Desc, bytes.NewReader(p.Manifest)); err != nil {
		return fmt.Errorf("error pushing manifest to repo %q: %v", p.Repo, err)
	}
	t.Fill(t.Value().FillPath(cue.MakePath(cue.Str("desc")), p.Desc))
	return nil
}

// getZip returns a zip archive consisting of all the files in ar
func getZip(files map[string]string) ([]byte, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no files found in zip archive")
	}
	names := make([]string, 0, len(files))
	for k := range files {
		names = append(names, k)
	}
	sort.Strings(names)

	var buf bytes.Buffer
	zipw := zip.NewWriter(&buf)
	for _, name := range names {
		w, err := zipw.Create(name)
		if err != nil {
			return nil, err
		}
		_, err = w.Write([]byte(files[name]))
		if err != nil {
			return nil, err
		}
	}
	if err := zipw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
