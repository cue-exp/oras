package registryclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

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
	if !isManifest(modDesc.MediaType) {
		return nil, fmt.Errorf("%v does not resolve to a manifest (media type is %q)", m, modDesc.MediaType)
	}
	var manifest ocispec.Manifest
	if err := fetchJSON(ctx, repo.Manifests(), modDesc, &manifest); err != nil {
		return nil, fmt.Errorf("cannot unmarshal manifest data: %v", err)
	}
	if len(manifest.Layers) < 1 {
		return nil, fmt.Errorf("module has no layers!")
	}
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

func (m *Module) Dependencies(ctx context.Context) (map[module.Version]Dependency, error) {
	if err := m.initConfig(ctx); err != nil {
		return nil, err
	}
	deps := make(map[module.Version]Dependency)
	for mname, resm := range m.cfg.ResolvedModules {
		mpath, mver, ok := strings.Cut(mname, "@")
		if !ok || mver == "" || !semver.IsValid(mver) {
			return nil, fmt.Errorf("bad module name %q found in module config", m)
		}
		mv := module.Version{
			Path:    mpath,
			Version: mver,
		}
		var desc ocispec.Descriptor
		found := false
		for _, layer := range m.manifest.Layers[1:] {
			if layer.Digest == resm.Digest {
				desc = layer
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("no layer found for module dependency %q", m)
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
