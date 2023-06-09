package modpush

import (
	"encoding/json"
	"strings"
	"github.com/cue-exp/oras"
)

// moduleArtifactType defines the artifact type of a CUE module.
// It's this that signifies at the top level that a given artifact contains
// a CUE module.
//
// TODO decide on what this should actually look like
moduleArtifactType: "application/vnd.cue.module.v1+json"

// moduleFileMediaType defines the media type of a module.cue file.
//
// TODO decide on what this should actually look like
// TODO should we gzip it?
moduleFileMediaType: "application/vnd.cue.modulefile.v1"

#module: {
	// moduleFile holds the contents of cue.mod/module.cue
	moduleFile!: _#ModuleFile

	// files holds all the files in the module, in txtar format.
	files!: [string]: string

	// deps holds all the modules that this module depends on.
	deps?: [... #module]

	// All subsequent fields are filled out automatically from the modules template.

	// path holds the module path.
	// TODO make this include the major version too.
	path!: string

	// pathVer holds the fully qualified module path including its minor version.
	// Filled out automatically.
	pathVer!: string

	// version holds the version of the module.
	// Filled out automatically.
	version!: string

	// repoActions is filled out automatically from the above fields
	// by the modules template.
	repoActions?: {
		scratchConfig: oras.#repoBlob
		layers!: [... oras.#repoBlob]
		manifest!: oras.#repoManifest
		tag!:      oras.#repoTag
	}
}

#modules: [#modver]: #module

#modver: =~"^[^@]+@[^@]+$"

#digest: =~"^sha256:.*"

modules: #modules

// This template derives all the task contents from the user-provided
// fields.
modules: [modNameVer=_]: {
	let _parts = strings.Split(modNameVer, "@")
	let _path = _parts[0]
	let _version = _parts[1]
	let _repoName = "cue/" + _path

	pathVer:     modNameVer
	path:        _path
	deps!:       _
	moduleFile!: _

	// We always include the module.cue file.
	files: "cue.mod/module.cue": json.Marshal(moduleFile)

	repoActions: {
		scratchConfig: oras.#repoBlob & {
			repo: _repoName
			desc: oras.scratchConfig
			desc: mediaType: moduleArtifactType
			source: {}
		}

		// Each dependency is represented as a layer.
		// We know which layer is which by attaching metadata to
		// the descriptors.
		//
		// The content of the module itself is always the
		// first layer.
		layers: [
			// The contents of this module. This must be layer 0.
			{
				repo: _repoName
				// TODO should we use a custom media type for this?
				desc: mediaType: "application/zip"
				source: files
			},
			// The module file for this module, extracted for easy access.
			{
				repo: _repoName
				desc: mediaType: moduleFileMediaType
				source: json.Marshal(moduleFile)
			},
			// All other dependencies of this module (order doesn't matter)
			for dep in deps {
				repo: _repoName

				// Take the module files (only) from the dependency.
				let depContent = dep.repoActions.layers[0]
				desc:   depContent.desc
				source: depContent.source

				// Add an annotation so that the client can know which layer
				// corresponds to which actual module version.
				desc: annotations: "works.cue.module": dep.pathVer
			},
		]

		// The manifest brings together the component pieces.
		manifest: {
			repo: _repoName
			manifest: {
				mediaType:    _
				artifactType: moduleArtifactType
				config:       scratchConfig.desc
				layers: [
					for layer in repoActions.layers {
						layer.desc
					},
				]
			}
		}
		// The tag gives a name to the whole thing.
		tag: {
			name:   _version
			"repo": _repoName
			desc:   repoActions.manifest.desc
		}
	}
}
