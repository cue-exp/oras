package oras

#repoBlob: {
	_oras:   "blob"
	repo!:   string
	desc!:   #descriptor
	source!: _
}

#repoTag: {
	_oras: "tag"
	repo!: string
	name!: string
	// desc must describe a manifest.
	desc!: #descriptor
}

#repoManifest: {
	_oras:     "manifest"
	repo!:     string
	manifest!: #manifest & {
		schemaVersion: _
	}
	desc?: #descriptor
}

#repoDump: {
	_oras: "dump"
	...
}
