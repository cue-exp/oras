package test

import (
	"encoding/json"
	"strings"
	"github.com/cue-exp/oras"
)

#module: {
	// path holds the module path.
	// TODO make this include the major version too.
	path!: string

	// pathVer holds the fully qualified module path including its minor version.
	pathVer!: string

	// version holds the version of the module.
	version!: string

	// moduleFile holds the contents of cue.mod/module.cue
	moduleFile!: _#ModuleFile

	// files holds all the files in the module, in txtar format.
	files!: [string]: string

	// deps holds all the modules that this module depends on.
	deps?: [... #module]

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
	moduleFile!: _#ModuleFile
}

#digest: =~"^sha256:.*"

modules: #modules

modules: [modNameVer=_]: {
	pathVer: modNameVer
	let _parts = strings.Split(modNameVer, "@")
	let _path = _parts[0]
	let _version = _parts[1]
	path: _path
	repoActions: tag: {
		name:   _version
		"repo": path
		digest: repoActions.manifest.desc.digest
	}
	deps!:  _
	moduleFile!: _
	files: "cue.mod/module.cue": json.Marshal(moduleFile)
	repoActions: {
		layers: [
			// The first blob is always the "self" module content.
			{
				repo: path
				desc: mediaType: "application/zip"
				source: files
			},
			// Each dependency is represented as a layer.
			// We know which layer is which because the
			// config blob holds that metadata.
			for dep in deps {
				repo: path
				let depSelf = dep.repoActions.layers[0]
				desc:   depSelf.desc
				source: depSelf.source
			},
		]
		config: {
			repo: path
			desc: mediaType: "application/json" // TODO custom module config json type
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
		manifest: {
			repo: path
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
