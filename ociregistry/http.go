package ociregistry

import (
	"context"
)

// HTTPRegistry provides access to the HTTP registry API, as defined [here].
// It implements [Interface].
//
// [here]:  https://github.com/opencontainers/distribution-spec/blob/main/spec.md
type HTTPRegistry struct {
	// unexported fields
}

type HTTPRegistryParams struct {
}

func NewHTTPRegistry(p HTTPRegistryParams) *HTTPRegistry {
	panic("TODO")
}

func (r *HTTPRegistry) Repositories(ctx context.Context) Iter[string] {
	panic("TODO")
}

func (r *HTTPRegistry) Tags(ctx context.Context, repo string) Iter[string] {
	panic("TODO")
}

func (r *HTTPRegistry) Referrers(ctx context.Context, repo string, digest Digest, artifactType string) Iter[Descriptor] {
	panic("TODO")
}

func (r *HTTPRegistry) GetManifest(ctx context.Context, repo string, digest Digest) (BlobReader, error) {
	panic("TODO")
}

func (r *HTTPRegistry) GetBlob(ctx context.Context, repo string, digest Digest) (BlobReader, error) {
	panic("TODO")
}

func (r *HTTPRegistry) PushBlob(ctx context.Context, repo string, c BlobReader, desc Descriptor) (Descriptor, error) {
	panic("TODO")
}

func (r *HTTPRegistry) PushManifest(ctx context.Context, repo string, c BlobReader, desc Descriptor) (Descriptor, error) {
	panic("TODO")
}

func (r *HTTPRegistry) Mount(ctx context.Context, repo string, fromRepo string, digest Digest) error {
	panic("TODO")
}

func (r *HTTPRegistry) Tag(ctx context.Context, repo string, manifestDigest Digest, tag string) error {
	panic("TODO")
}

func (r *HTTPRegistry) DeleteBlob(ctx context.Context, repo string, digest Digest) error {
	panic("TODO")
}

func (r *HTTPRegistry) DeleteManifest(ctx context.Context, repo string, digest Digest) error {
	panic("TODO")
}

func (r *HTTPRegistry) DeleteTag(ctx context.Context, repo string, name string) error {
	panic("TODO")
}
