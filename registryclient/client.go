package registryclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/mod/module"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
)

type Client struct {
	remote *remote.Registry
}

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

func (c *Client) repo(ctx context.Context, m module.Version) (registry.Repository, error) {
	return c.remote.Repository(ctx, "cue/"+m.Path)
}

func (c *Client) GetModule(ctx context.Context, m module.Version) (*Module, error) {
	repo, err := c.repo(ctx, m)
	if err != nil {
		return nil, fmt.Errorf("cannot determine remote repository: %v", err)
	}

	modDesc, err := repo.Manifests().Resolve(ctx, m.Version)
	if err != nil {
		// TODO not-found error
		return nil, fmt.Errorf("cannot resolve %v: %v", m, err)
	}
	if !isManifest(modDesc.MediaType) {
		return nil, fmt.Errorf("%v does not resolve to a manifest (media type is %q)", m, modDesc.MediaType)
	}
	var manifest ocispec.Manifest
	if err := fetchJSON(ctx, repo.Manifests(), modDesc, &manifest); err != nil {
		return nil, fmt.Errorf("cannot unmarshal manifest data: %v", err)
	}
	return &Module{
		repo:     repo,
		manifest: manifest,
	}, nil
}

func (c *Client) ModuleVersions(ctx context.Context, m string) ([]string, error) {
	panic("unimplemented")
}

type moduleConfig struct {
	ResolvedModules map[string]resolvedModule `json:"resolvedModules"`
	ModuleFile      json.RawMessage           `json:"moduleFile"`
}

type resolvedModule struct {
	Digest string `json:"digest"`
}

type Module struct {
	repo     registry.Repository
	manifest ocispec.Manifest

	initOnce sync.Once
	cfg      *moduleConfig
	cfgErr   error
}

func (m *Module) initConfig(ctx context.Context) error {
	m.initOnce.Do(func() {
		m.cfgErr = fetchJSON(ctx, m.repo, m.manifest.Config, &m.cfg)
	})
	return m.cfgErr
}

func (m *Module) ModuleFile(ctx context.Context) ([]byte, error) {
	if err := m.initConfig(ctx); err != nil {
		return nil, err
	}
	return m.cfg.ModuleFile, nil
}

func (m *Module) GetZip(ctx context.Context) (io.ReadCloser, error) {
	panic("unimplemented")
}

func (m *Module) Dependencies() (map[module.Version]*Dependency, error) {
	panic("unimplemented")
}

type Dependency struct {
	// TODO
}

func (d *Dependency) Version() module.Version {
	panic("unimplemented")
}

func (d *Dependency) GetZip() (io.ReadCloser, error) {
	panic("unimplemented")
}

func fetchJSON(ctx context.Context, from content.Fetcher, desc ocispec.Descriptor, dst any) error {
	if !isJSON(desc.MediaType) {
		return fmt.Errorf("expected JSON media type but %q does not look like JSON", desc.MediaType)
	}
	r, err := from.Fetch(ctx, desc)
	if err != nil {
		return fmt.Errorf("cannot fetch content: %v", err)
	}
	defer r.Close()
	dec := json.NewDecoder(r)
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("cannot decode content into %T: %v", dst, err)
	}
	return nil
}

func isManifest(mediaType string) bool {
	return mediaType == "application/vnd.oci.image.manifest.v1+json"
}

// isJSON reports whether the given media type has JSON as an underlying encoding.
// TODO this is a guess. There's probably a more correct way to do it.
func isJSON(mediaType string) bool {
	return strings.HasSuffix(mediaType, "+json") || strings.HasSuffix(mediaType, "/json")
}
