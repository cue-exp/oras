package registryclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rogpeppe/go-internal/semver"
	"golang.org/x/mod/module"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
)

type Client struct {
	remote *remote.Registry
}

const (
	moduleArtifactType  = "application/vnd.cue.module.v1+json"
	moduleFileMediaType = "application/vnd.cue.modulefile.v1"
	moduleAnnotation    = "works.cue.module"
)

func New(registry string) (*Client, error) {
	reg, err := remote.NewRegistry(registry)
	if err != nil {
		return nil, err
	}
	// TODO is this in the right place?
	if strings.HasPrefix(registry, "localhost:") {
		reg.PlainHTTP = true
	}
	return &Client{
		remote: reg,
	}, nil
}

func (c *Client) repo(ctx context.Context, modPath string) (registry.Repository, error) {
	return c.remote.Repository(ctx, "cue/"+modPath)
}

func (c *Client) GetModule(ctx context.Context, m module.Version) (*Module, error) {
	repo, err := c.repo(ctx, m.Path)
	if err != nil {
		return nil, fmt.Errorf("cannot determine remote repository: %v", err)
	}

	modDesc, err := repo.Manifests().Resolve(ctx, m.Version)
	if err != nil {
		// TODO not-found error
		return nil, fmt.Errorf("cannot resolve %v: %v", m, err)
	}
	var manifest ocispec.Manifest
	if err := fetchJSON(ctx, repo.Manifests(), modDesc, &manifest); err != nil {
		return nil, fmt.Errorf("cannot unmarshal manifest data: %v", err)
	}
	if !isModule(&manifest) {
		return nil, fmt.Errorf("%v does not resolve to a manifest (media type is %q)", m, modDesc.MediaType)
	}
	// TODO check type of manifest too.
	if n := len(manifest.Layers); n < 2 {
		return nil, fmt.Errorf("not enough blobs found in module manifest; need at least 2, got %d", n)
	}
	if !isModuleFile(manifest.Layers[1]) {
		return nil, fmt.Errorf("unexpected media type %q for module file blob", manifest.Layers[1].MediaType)
	}
	// TODO check that all other blobs are of the expected type (application/zip)
	// and that dependencies have the expected attribute.
	return &Module{
		client:   c,
		repo:     repo,
		manifest: manifest,
	}, nil
}

func (c *Client) ModuleVersions(ctx context.Context, m string) ([]string, error) {
	repo, err := c.repo(ctx, m)
	if err != nil {
		return nil, fmt.Errorf("cannot determine remote repository: %v", err)
	}
	var versions []string
	if err := repo.Tags(ctx, "", func(tags []string) error {
		for _, tag := range tags {
			if semver.IsValid(tag) {
				versions = append(versions, tag)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return versions, nil
}

type moduleConfig struct {
	ResolvedModules map[string]resolvedModule `json:"resolvedModules"`
	ModuleFile      json.RawMessage           `json:"moduleFile"`
}

type resolvedModule struct {
	Digest digest.Digest `json:"digest"`
}

type Module struct {
	client   *Client
	repo     registry.Repository
	manifest ocispec.Manifest
}

func (m *Module) ModuleFile(ctx context.Context) ([]byte, error) {
	return fetchBytes(ctx, m.repo, m.manifest.Layers[1])
}

func (m *Module) GetZip(ctx context.Context) (io.ReadCloser, error) {
	panic("unimplemented")
}

func (m *Module) Dependencies(ctx context.Context) (map[module.Version]Dependency, error) {
	deps := make(map[module.Version]Dependency)
	for _, desc := range m.manifest.Layers[2:] {
		mname, ok := desc.Annotations[moduleAnnotation]
		if !ok {
			return nil, fmt.Errorf("no %s annotation found for blob", moduleAnnotation)
		}
		mpath, mver, ok := strings.Cut(mname, "@")
		if !ok || mver == "" || !semver.IsValid(mver) {
			return nil, fmt.Errorf("bad module name %q found in module config", m)
		}
		mv := module.Version{
			Path:    mpath,
			Version: mver,
		}
		deps[mv] = Dependency{
			client:  m.client,
			version: mv,
			desc:    desc,
		}
	}
	return deps, nil
}

type Dependency struct {
	client  *Client
	version module.Version
	desc    ocispec.Descriptor
}

func (d Dependency) Version() module.Version {
	return d.version
}

func (d Dependency) GetZip() (io.ReadCloser, error) {
	panic("unimplemented")
}

func fetchJSON(ctx context.Context, from content.Fetcher, desc ocispec.Descriptor, dst any) error {
	if !isJSON(desc.MediaType) {
		return fmt.Errorf("expected JSON media type but %q does not look like JSON", desc.MediaType)
	}
	data, err := fetchBytes(ctx, from, desc)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("cannot decode %s content into %T: %v", desc.MediaType, dst, err)
	}
	return nil
}

func fetchBytes(ctx context.Context, from content.Fetcher, desc ocispec.Descriptor) ([]byte, error) {
	r, err := from.Fetch(ctx, desc)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch content: %v", err)
	}
	defer r.Close()
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("cannot read content: %v", err)
	}
	return data, err
}

func isModule(m *ocispec.Manifest) bool {
	// TODO check m.ArtifactType too when that's defined?
	// See https://github.com/opencontainers/image-spec/blob/main/manifest.md#image-manifest-property-descriptions
	return m.Config.MediaType == moduleArtifactType
}

func isModuleFile(desc ocispec.Descriptor) bool {
	return desc.ArtifactType == moduleFileMediaType ||
		desc.MediaType == moduleFileMediaType
}

// isJSON reports whether the given media type has JSON as an underlying encoding.
// TODO this is a guess. There's probably a more correct way to do it.
func isJSON(mediaType string) bool {
	return strings.HasSuffix(mediaType, "+json") || strings.HasSuffix(mediaType, "/json")
}
