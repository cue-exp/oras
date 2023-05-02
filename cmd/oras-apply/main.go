package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote"

	"github.com/cue-exp/oras/orasflow"
)

const (
	debug          = true
	singleThreaded = true
)

var (
	nflag      = flag.Bool("n", false, "print what we're doing but do not actually do anything")
	scriptFlag = flag.Bool("script", false, "generate command line script")
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
	ctx := context.Background()
	registry, err := getRegistry(ctx)
	if err != nil {
		return err
	}
	if err := orasflow.Apply(ctx, v, registry); err != nil {
		return err
	}
	if r, ok := registry.(*loggingRegistry); ok {
		r.dumpMermaid()
	}
	return nil
}

func getRegistry(ctx context.Context) (orasflow.Registry, error) {
	if *nflag {
		return newLoggingRegistry(), nil
	}

	reg := os.Getenv("OCI_REGISTRY")
	if reg == "" {
		reg = "localhost:5000" // TODO lose this default
	}
	if *scriptFlag {
		return newScriptRegistry(reg), nil
	}
	registry, err := remote.NewRegistry(reg)
	if err != nil {
		return nil, fmt.Errorf("cannot make registry instance: %v", err)
	}
	registry.PlainHTTP = true
	if err := registry.Ping(ctx); err != nil {
		return nil, fmt.Errorf("cannot ping registry: %v", err)
	}
	return orasflow.RegistryFromRemote(registry), nil
}

type arc struct {
	n1, n2 string
}

func newLoggingRegistry() *loggingRegistry {
	return &loggingRegistry{
		pushed:       make(map[string]string),
		repositories: make(map[string]int),
	}
}

type loggingRegistry struct {
	mu           sync.Mutex
	pushed       map[string]string // map from node name to node content
	arcs         []arc
	repositories map[string]int
}

//	r1:sha256:24433[example.com/foo text/plain]
//	r2:sha256:335[arble.com application/zip]
//	r1:sha256:335-->|.foo.bar|r2sha256:24433

var refPat = regexp.MustCompile("sha256:[a-f0-9]{64,}")

func init() {
	refPat.Longest()
}

func (r *loggingRegistry) dumpMermaid() {
	var buf strings.Builder
	printf := func(f string, a ...any) {
		fmt.Fprintf(&buf, f, a...)
	}
	printf("flowchart LR\n")
	for ref, name := range r.pushed {
		printf("\t%s[%s]\n", ref, name)
	}
	for _, a := range r.arcs {
		printf("\t%s-->%s\n", a.n1, a.n2)
	}
}

func (r *loggingRegistry) Push(ctx context.Context, repoName string, desc ocispec.Descriptor, content io.Reader) error {
	log.Printf("--- push")
	r.addRefs(repoName, desc, content)
	return nil
}

func (r *loggingRegistry) PushManifest(ctx context.Context, repoName string, desc ocispec.Descriptor, content io.Reader) error {
	log.Printf("--- pushManifest")
	r.addRefs(repoName, desc, content)
	return nil
}

func (r *loggingRegistry) Tag(ctx context.Context, repoName string, desc ocispec.Descriptor, reference string) error {
	log.Printf("--- tag")
	r.addArc("tag:%s", r.repoDigest(repoName, desc.Digest))
	return nil
}

func (r *loggingRegistry) addRefs(repoName string, desc ocispec.Descriptor, content io.Reader) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !isJSON(desc.MediaType) && !isText(desc.MediaType) {
		return
	}
	data, err := ioutil.ReadAll(content)
	if err != nil {
		panic(fmt.Errorf("cannot read content: %v", err))
	}
	r.addDesc(repoName, desc)
	from := r.repoDigest(repoName, desc.Digest)
	for _, ref := range r.scanText(repoName, string(data)) {
		r.addArc(from, ref)
	}
}

func (r *loggingRegistry) Dump(ctx context.Context, stuff json.RawMessage) {
}

func (r *loggingRegistry) addDesc(repoName string, desc ocispec.Descriptor) {
	// TODO check for media type clashes
	r.pushed[r.repoDigest(repoName, desc.Digest)] = repoName + " " + desc.MediaType
}

func (r *loggingRegistry) scanText(repoName, text string) []string {
	possibleRefs := refPat.FindAllString(text, -1)
	log.Printf("XXX matches: %q", possibleRefs)
	refs := make([]string, 0, len(possibleRefs))
	for _, ref := range possibleRefs {
		idRef := r.repoDigest(repoName, digest.Digest(ref))
		if _, ok := r.pushed[idRef]; ok {
			refs = append(refs, idRef)
		} else {
			log.Printf("XXXX no match for %v", idRef)
		}
	}
	log.Printf("--- scanned %s -> %v", text, refs)
	return refs
}

func (r *loggingRegistry) repoDigest(repoName string, digest digest.Digest) string {
	return fmt.Sprintf("r%d:%s", r.repoID(repoName), digest)
}

func (r *loggingRegistry) repoID(repoName string) int {
	if r.repositories == nil {
		r.repositories = make(map[string]int)
	}
	if id := r.repositories[repoName]; id > 0 {
		return id
	}
	r.repositories[repoName] = len(r.repositories) + 1
	return len(r.repositories)
}

func (r *loggingRegistry) addArc(n1, n2 string) {
	r.arcs = append(r.arcs, arc{n1, n2})
}

func isJSON(mediaType string) bool {
	return strings.HasSuffix(mediaType, "+json") || strings.HasSuffix(mediaType, "/json")
}

func isText(mediaType string) bool {
	return mediaType == "text/plain"
}
