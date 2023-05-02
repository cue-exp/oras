package oras

import (
	"encoding/base64"
	"crypto/sha256"
	"encoding/hex"
)

// A "scratch" configuration descriptor.
// https://github.com/opencontainers/image-spec/blob/main/manifest.md#example-of-a-scratch-config-or-layer-descriptor
scratchConfig: #descriptor & {
	let content = "{}"
	mediaType: *"application/vnd.oci.scratch.v1+json" | string
	size:      len(content)
	data:      '\(base64.Encode(null, content))'
	digest:    "sha256:\(hex.Encode(sha256.Sum256(content)))"
}
