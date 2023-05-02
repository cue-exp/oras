package mediatypes

type: "application/vnd.oci.image.manifest.v1+json": {
	// schemaVersion specifies the image manifest schema version.
	// For this version of the specification, this MUST be 2
	// to ensure backward compatibility with older versions of Docker.
	// The value of this field will not change.
	// This field may be removed in a future version of the specification.
	schemaVersion!: 2

	// mediaType is reserved for use to maintain compatibility.
	// This field should be set for backward compatibility.
	// Its usage differs from the descriptor use of mediaType.
	mediaType?: "application/vnd.oci.image.manifest.v1+json"

	// This field contains the type of an artifact when the manifest is used for an artifact.
	// If defined, the value must comply with RFC 6838,
	// including the naming requirements in its section 4.2,
	// and MAY be registered with IANA.
	artifactType?: string
	if mediaType != _|_ {
		artifactType!: _
	}

	// config references a configuration object for a container by digest.
	// Manifests concerned with portability should use the media type
	// [application/vnd.oci.image.config.v1+json].
	//
	// [application/vnd.oci.image.config.v1+json]: https://github.com/opencontainers/image-spec/blob/v1.0.2/config.md
	config!: #descriptor

	// layers holds the list of blobs that comprise the content of the manifest item.
	// The array must have the base layer at index 0.
	// Subsequent layers must then follow in stack order (i.e. from layers[0] to layers[len(layers)-1]).
	// The final filesystem layout must match the result of applying the layers to an empty directory.
	// The ownership, mode, and other attributes of the initial empty directory are unspecified.
	//
	// Manifests concerned with portability should use one of the following media types.
	// - application/vnd.oci.image.layer.v1.tar
	// - application/vnd.oci.image.layer.v1.tar+gzip
	// - application/vnd.oci.image.layer.nondistributable.v1.tar
	// - application/vnd.oci.image.layer.nondistributable.v1.tar+gzip
	layers!: [... #descriptor]

	// subject specifies a descriptor of another manifest.
	// This value, used by the referrers API,
	// indicates a relationship to the specified manifest.
	subject?: #descriptor

	// annotations holds arbitrary metadata for the image manifest.
	// It must use the [annotation rules].
	//
	// See [Pre-Defined Annotation Keys].
	//
	// [annotation rules]: https://github.com/opencontainers/image-spec/blob/v1.1.0-rc2/annotations.md#rules
	// [Pre-Defined Annotation Keys]: https://github.com/opencontainers/image-spec/blob/v1.1.0-rc2/annotations.md#pre-defined-annotation-keys
	annotations?: [string]: string
}
