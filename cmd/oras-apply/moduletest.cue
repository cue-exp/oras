package test

import (
	"encoding/json"
	"strings"
	"github.com/cue-exp/oras"
)

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
		layers!: [... oras.#repoBlob]
		config!: oras.#repoBlob & {
			source: #moduleConfig
		}
		manifest!: oras.#repoManifest
		tag!:      oras.#repoTag
	}
}

#modules: [#modver]: #module

#modver: =~"^[^@]+@[^@]+$"

// #moduleConfig defines the configuration blob uploaded
// as part of a module.
#moduleConfig: {
	resolvedModules: [#modver]: {
		digest!: #digest
	}
	// This should probably be a string so that it can hold
	// comments too.
	moduleFile!: _#ModuleFile
}

#digest: =~"^sha256:.*"

modules: #modules

// moduleConfigMediaType defines the media type of the config object
// pointed to by the module manifest.
//
// TODO decide on what this should actually look like
moduleConfigMediaType: "application/cue.module.config.v1+json"

// This template derives all the task contents from the user-provided
// fields.
modules: [modNameVer=_]: {
	let _parts = strings.Split(modNameVer, "@")
	let _path = _parts[0]
	let _version = _parts[1]
	let _repoName = "cue/" + _path

	pathVer: modNameVer
	path: _path
	deps!:       _
	moduleFile!: _

	// We always include the module.cue file.
	files: "cue.mod/module.cue": json.Marshal(moduleFile)

	repoActions: {
		tag: {
			name:   _version
			"repo": _repoName
			desc: repoActions.manifest.desc
		}
		// Each dependency is represented as a layer.
		// We know which layer is which because the
		// config blob holds that metadata.
		//
		// The content of the module itself is always the
		// first layer.
		layers: [
			// "self" module content.
			{
				repo: _repoName
				desc: mediaType: "application/zip"
				source: files
			},

			// All other dependencies.
			for dep in deps {
				repo: _repoName
				let depSelf = dep.repoActions.layers[0]
				desc:   depSelf.desc
				source: depSelf.source
			},
		]

		// The config object holds JSON metadata that tells
		// the reader which layer corresponds to which dependency,
		// and also holds the contents of the module.cue file for
		// easy access.
		config: {
			repo: _repoName
			desc: mediaType: moduleConfigMediaType
			source: {
				resolvedModules: {
					for dep in deps {
						(dep.pathVer): {
							digest: dep.repoActions.layers[0].desc.digest
						}
					}
				}
				"moduleFile": moduleFile
			}
		}

		// The manifest brings it all together.
		manifest: {
			repo: _repoName
			manifest: {
				config: repoActions.config.desc
				layers: [
					for layer in repoActions.layers {
						layer.desc
					},
				]
			}
		}
	}
}
