package ociregistry

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Interface defines a generic interface to a single OCI registry.
// It does not support cross-registry operations: all methods are
// directed to the receiver only.
type Interface interface {
	Writer
	Reader
	Deleter
}

type (
	Digest     = digest.Digest
	Descriptor = ocispec.Descriptor
	Manifest   = ocispec.Manifest
)

type Reader interface {
	GetBlob(ctx context.Context, repo string, digest Digest) (BlobReader, error)
	GetManifest(ctx context.Context, repo string, digest Digest) (BlobReader, error)
	GetTag(ctx context.Context, repo string, tagName string) (BlobReader, error)
}

type Writer interface {
	PushBlob(ctx context.Context, repo string, c BlobReader, desc Descriptor) (Descriptor, error)
	PushManifest(ctx context.Context, repo string, c BlobReader, desc Descriptor) (Descriptor, error)
	Mount(ctx context.Context, repo string, fromRepo string, digest Digest) error
	Tag(ctx context.Context, repo string, digest Digest, tag string) error
}

type Deleter interface {
	DeleteBlob(ctx context.Context, repo string, digest Digest) error
	DeleteManifest(ctx context.Context, repo string, digest Digest) error
	DeleteTag(ctx context.Context, repo string, name string) error
}

type Lister interface {
	Repositories(ctx context.Context) Iter[string]
	Tags(ctx context.Context, repo string) Iter[string]
	Referrers(ctx context.Context, repo string, digest Digest, artifactType string) Iter[Descriptor]
}

// BlobReader provides the contents of a given blob or manifest.
type BlobReader interface {
	Descriptor() Descriptor
	Open() io.ReadCloser
	OpenRange(p0, p1 int64) io.ReadCloser
}

func BytesBlob(data []byte, contentType string) BlobReader {
	panic("TODO")
}

func FileBlob(f *os.File, contentType string) BlobReader {
	panic("TODO")
}

func ManifestBlob(m *Manifest) BlobReader {
	panic("TODO")
}

type ReadWriter interface {
	Reader
	Writer
}

// Client provides general operations that can span multiple registries.
// The "reference" string taken by most methods includes a host name
// that's used to locate the
type Client struct {
	resolveRegistry func(host string) (Interface, error)
}

func (c *Client) Copy(ctx context.Context, dstRef, srcRef string, includeReferrers bool) error {
	panic("TODO")
}
func (c *Client) Push(ctx context.Context, dstRef string, r BlobReader) error {
	panic("TODO")
}
func (c *Client) GetBlob(ctx context.Context, ref string) (BlobReader, error) {
	panic("TODO")
}
func (c *Client) GetManifest(ctx context.Context, ref string) (BlobReader, error) {
	panic("TODO")
}

// Copy copies the manifest and all its references from src to dst.
func Copy(digest Digest, dst ReadWriter, src Reader) (Descriptor, error) {
	panic("TODO")
}

// Serve returns an HTTP handler that provides a handler for the OCI registry API
// using r as its backing.
func Serve(r Interface) http.Handler {
	panic("TODO")
}
