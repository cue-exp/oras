package oras

// manifest defines the [application/vnd.oci.image.manifest.v1+json] media type. For the media type(s) that this is compatible with see the [matrix].
//
// [matrix]: https://github.com/opencontainers/image-spec/blob/main/media-types.md#compatibility-matrix
#manifest: {
	// schemaVersion specifies the image manifest schema version.
	// For this version of the specification, this MUST be 2
	// to ensure backward compatibility with older versions of Docker.
	// The value of this field will not change.
	// This field may be removed in a future version of the specification.
	schemaVersion!: 2

	// mediaType is reserved for use to maintain compatibility.
	// When used, this field contains the media type of this document,
	// which differs from the descriptor use of mediaType.
	mediaType?: string

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

// #descriptor describes the "application/vnd.oci.descriptor.v1+json" media type.
//
// - An OCI image consists of several different components, arranged in a Merkle Directed Acyclic Graph (DAG).
// - References between components in the graph are expressed through Content Descriptors.
// - A Content Descriptor (or simply Descriptor) describes the disposition of the targeted content.
// - A Content Descriptor includes the type of the content, a content identifier (digest), and the byte-size of the raw content.
// - Descriptors should be embedded in other formats to securely reference external content.
// - Other formats should use descriptors to securely reference external content.
#descriptor: {
	// metaType holds the media type of the referenced content.
	// Values must comply with [RFC 6838],
	// including the [naming requirements] in its section 4.2.
	// The OCI image specification defines [several of its own MIME types]
	// for resources defined in the specification.
	//
	// JSON content shiuld be serialized as [canonical JSON].
	//
	// [RFC 6838]: https://tools.ietf.org/html/rfc6838
	// [naming requirements]: https://tools.ietf.org/html/rfc6838#section-4.2
	// [several of its own MIME types]: https://github.com/opencontainers/image-spec/blob/v1.0.1/media-types.md
	// [canonical JSON]: https://wiki.laptop.org/go/Canonical_JSON
	mediaType!: string

	//  digest holds the digest of the targeted content,
	// conforming to the requirements outlined in [Digests].
	// Retrieved content should be verified against this digest
	// when consumed via untrusted sources.
	//
	// NOTE: this is required on the wire and is optional only
	// so we can unmarshal it for the flow task.
	//
	// [Digests]: https://github.com/opencontainers/image-spec/blob/v1.0.1/descriptor.md#digests
	digest?: string

	// size specifies the size, in bytes, of the raw content.
	// This property exists so that a client will have
	// an expected size for the content before processing.
	// If the length of the retrieved content does not match
	// the specified length,
	// the content should not be trusted.
	//
	// NOTE: this is required on the wire and is optional only
	// so we can unmarshal it for the flow task.
	size?: int64

	// urls specifies a list of URIs from which this object may be downloaded.
	// Each entry must conform to [RFC 3986]. Entries should use the
	// http and https schemes, as defined in [RFC 7230].
	//
	// [RFC 3986]: https://tools.ietf.org/html/rfc3986
	// [RFC 7230]: https://tools.ietf.org/html/rfc7230
	urls?: [... string]

	/// annotation contains arbitrary metadata for this descriptor. It must use the [annotation rules].
	//
	// [annotation rules]:
	annotations?: [string]: string

	if mediaType == "application/vnd.oci.image.manifest.v1+json" {
		// TODO https://github.com/opencontainers/image-spec/blob/v1.0.1/image-index.md#image-index-property-descriptions
		platform!: _
	}

	// data contains an embedded representation of the referenced content.
	// Values must conform to the Base 64 encoding, as defined in [RFC 4648].
	// The decoded data must be identical to the referenced content and
	// should be verified against the digest and size fields by content consumers.
	// See [Embedded Content] for when this is appropriate.
	//
	// [RFC 4648]: 	https://tools.ietf.org/html/rfc4648
	// [Embedded Content]: https://github.com/opencontainers/image-spec/blob/main/descriptor.md#embedded-content
	data?: string

	// artifactType contains the type of an artifact when the descriptor points to an artifact.
	// This is the value of the config descriptor "mediaType"
	// when the descriptor references an image manifest.
	// If defined, the value must comply with RFC 6838,
	// including the naming requirements in its section 4.2,
	// and may be registered with [IANA].
	//
	// [RFC 6838]: https://tools.ietf.org/html/rfc6838
	// [IANA]: https://www.iana.org/assignments/media-types/media-types.xhtml
	artifactType?: string
}
