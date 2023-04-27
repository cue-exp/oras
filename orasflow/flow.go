package orasflow

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/tools/flow"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	debug          = true
	singleThreaded = true
)

const orasPkg = "github.com/cue-exp/oras"

var orasField = cue.MakePath(cue.Hid("_oras", orasPkg))

func Apply(ctx context.Context, v cue.Value, registry Registry) error {
	a := &applier{
		cueCtx:   v.Context(),
		registry: registry,
	}
	ctl := flow.New(nil, v, a.getTask)
	if err := ctl.Run(ctx); err != nil {
		return fmt.Errorf("error running flow: %v", errors.Details(err, nil))
	}
	return nil
}

type applier struct {
	cueCtx   *cue.Context
	registry Registry
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
		return wrapRunner(a.pushBlob), nil
	case "tag":
		return wrapRunner(a.pushTag), nil
	case "manifest":
		return wrapRunner(a.pushManifest), nil
	default:
		return nil, fmt.Errorf("unknown _oras field value %q", s)
	}
}

type blobPush struct {
	Desc   ocispec.Descriptor `json:"desc"`
	Repo   string             `json:"repo,omitempty"`
	Source json.RawMessage    `json:"source"`
}

func (a *applier) pushBlob(t *flow.Task) error {
	ctx := t.Context()
	var p blobPush
	if err := t.Value().Decode(&p); err != nil {
		return fmt.Errorf("cannot decode blob spec from path %v (%v): %v", t.Path(), t.Value(), err)
	}
	var sourceData []byte
	switch mtype := p.Desc.MediaType; {
	case isJSON(mtype):
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
	default:
		var s string
		if err := json.Unmarshal(p.Source, &s); err != nil {
			return fmt.Errorf("invalid source %#v for media type %q (want string)", p.Source, mtype)
		}
		sourceData = []byte(s)
	}
	p.Desc.Digest = digest.FromBytes(sourceData)
	p.Desc.Size = int64(len(sourceData))

	prettySource, _ := json.MarshalIndent(p.Source, "\t", "\t")
	logf("%v: push %s to %s -> %s\n\tsource: %s", t.Path(), p.Desc.MediaType, p.Repo, p.Desc.Digest, prettySource)

	if err := a.registry.Push(ctx, p.Repo, p.Desc, bytes.NewReader(sourceData)); err != nil {
		return fmt.Errorf("error pushing blob to repo %q: %v", p.Repo, err)
	}
	t.Fill(t.Value().FillPath(cue.MakePath(cue.Str("desc"), cue.Str("digest")), p.Desc.Digest))
	t.Fill(t.Value().FillPath(cue.MakePath(cue.Str("desc"), cue.Str("size")), p.Desc.Size))
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

	p.Desc.MediaType = ocispec.MediaTypeArtifactManifest
	p.Desc.Digest = digest.FromBytes(p.Manifest)
	p.Desc.Size = int64(len(p.Manifest))

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v: push manifest to %s -> %s\n", t.Path(), p.Repo, p.Desc.Digest)
	fmt.Fprintf(&buf, "\tsource: ")
	json.Indent(&buf, p.Manifest, "\t", "\t")
	logf("%s", buf.String())

	if err := a.registry.PushManifest(ctx, p.Repo, p.Desc, bytes.NewReader(p.Manifest)); err != nil {
		return fmt.Errorf("error pushing manifest to repo %q: %v", p.Repo, err)
	}
	t.Fill(t.Value().FillPath(cue.MakePath(cue.Str("desc")), p.Desc))
	return nil
}

type tagPush struct {
	Repo string             `json:"repo,omitempty"`
	Name string             `json:"name"`
	Desc ocispec.Descriptor `json:"desc"`
}

func (a *applier) pushTag(t *flow.Task) error {
	ctx := t.Context()
	var p tagPush
	if err := t.Value().Decode(&p); err != nil {
		return fmt.Errorf("cannot decode manifest spec from path %v (%v): %v", t.Path(), t.Value(), err)
	}
	logf("%v: push tag %s:%s", t.Path(), p.Repo, p.Name)
	if err := a.registry.Tag(ctx, p.Repo, p.Desc, p.Name); err != nil {
		return fmt.Errorf("cannot create tag %q: %v", p.Name, err)
	}
	return nil
}

// getZip returns a zip archive consisting of all the given files, keyed by filename.
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
		if _, err := w.Write([]byte(files[name])); err != nil {
			return nil, err
		}
	}
	if err := zipw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// isJSON reports whether the given media type has JSON as an underlying encoding.
// TODO this is a guess. There's probably a more correct way to do it.
func isJSON(mediaType string) bool {
	return strings.HasSuffix(mediaType, "+json") || strings.HasSuffix(mediaType, "/json")
}

var globalMutex sync.Mutex
var prevTime = time.Now()

func wrapRunner(f flow.RunnerFunc) flow.RunnerFunc {
	if !singleThreaded {
		return f
	}
	return func(t *flow.Task) error {
		globalMutex.Lock()
		logf("-- run %v (%v){", t.Path(), time.Since(prevTime))
		defer logf("}")
		defer globalMutex.Unlock()
		err := f(t)
		if err != nil {
			logf("-> error %v", err)
		}
		prevTime = time.Now()
		return err
	}
}

func logf(f string, a ...any) {
	if debug {
		fmt.Println(fmt.Sprintf(f, a...))
	}
}
