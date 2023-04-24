package oras

#repoBlob: {
	_oras: "blob"
	repo!: string
	desc!: #descriptor
	source!: _
}

#repoTag: {
	_oras: "tag"
	repo!: string
	name!: string
	// Must be digest of a manifest.
	digest!: string
}

#repoManifest: {
	_oras: "manifest"
	repo!: string
	manifest!: #manifest
	desc?: #descriptor
}
