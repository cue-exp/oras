package test

import (
	"strings"
	"github.com/cue-exp/oras"
)

#module: {
	path!: string
	pathVer!: string
	version!: string
	files!: string	// txtar content
	deps?: [... #module]

	repo?: {
		layers!: [... oras.#repoBlob]
		config!: oras.#repoBlob & {
			source: #moduleConfig
		}
		manifest!: oras.#repoManifest
		tag!: oras.#repoTag
	}
}

#modules: [#modver]: #module

#modver: =~ "^[^@]+@[^@]+$"

#moduleConfig: {
	resolvedModules: [#modver]: {
		digest!: #digest
	}
}

#digest: =~ "^sha256:.*"

modules: #modules

modules: [modNameVer=_]: {
	pathVer: modNameVer
	let _parts = strings.Split(modNameVer, "@")
	let _path = _parts[0]
	let _version = _parts[1]
	path: _path
	repo: tag: {
		name: _version
		"repo": path
		digest: repo.manifest.desc.digest
	}
	deps!: _
	files!: _
	repo: layers: [
		// The first blob is always the "self" module content.
		{
			repo: path
			desc: mediaType: "application/zip"
			source: files
		},
		for dep in deps {
			repo: path
			let depSelf = dep.repo.layers[0]
			desc: depSelf.desc
			source: depSelf.source
		}
	]
	repo: config: {
		repo: path
		desc: mediaType: "application/json"		// TODO custom module config json type
		source: {
			resolvedModules: {
				for dep in deps {
					(dep.pathVer): {
						digest: dep.repo.layers[0].desc.digest
					}
				}
			}
		}
	}
	repo: manifest: {
		"repo": path
		manifest: {
			config: repo.config.desc
			layers: [
				for layer in repo.layers {
					layer.desc
				}
			]
		}
	}
}
