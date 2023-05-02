package orasflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote"
)

func RegistryFromRemote(r *remote.Registry) Registry {
	return registryShim{r}
}

// Registry implements one-level interface API to the oras registry API.
// It's defined like this so that the required API surface area is clear
// and it's easier to implement.
type Registry interface {
	Push(ctx context.Context, repoName string, desc ocispec.Descriptor, content io.Reader) error
	PushManifest(ctx context.Context, repoName string, desc ocispec.Descriptor, content io.Reader) error
	Tag(ctx context.Context, repoName string, desc ocispec.Descriptor, reference string) error
	Dump(ctx context.Context, stuff json.RawMessage)
}

type registryShim struct {
	r *remote.Registry
}

func (r registryShim) Push(ctx context.Context, repoName string, desc ocispec.Descriptor, content io.Reader) error {
	repo, err := r.r.Repository(ctx, repoName)
	if err != nil {
		return fmt.Errorf("cannot make repository from %q: %v", repoName, err)
	}
	return repo.Push(ctx, desc, content)
}

func (r registryShim) PushManifest(ctx context.Context, repoName string, desc ocispec.Descriptor, content io.Reader) error {
	repo, err := r.r.Repository(ctx, repoName)
	if err != nil {
		return fmt.Errorf("cannot make repository from %q: %v", repoName, err)
	}
	return repo.Manifests().Push(ctx, desc, content)
}

func (r registryShim) Tag(ctx context.Context, repoName string, desc ocispec.Descriptor, reference string) error {
	repo, err := r.r.Repository(ctx, repoName)
	if err != nil {
		return fmt.Errorf("cannot make repository from %q: %v", repoName, err)
	}
	return repo.Tag(ctx, desc, reference)
}

func (r registryShim) Dump(ctx context.Context, stuff json.RawMessage) {
}
