package test

import "github.com/cue-exp/oras"

host: *"localhost:5000" | string

// We're pushing everything to the same repository
entities: [_]: repo: "blobmanifest-test"
entities: {
	unreferencedBlob: oras.#repoBlob & {
		desc: mediaType: "text/plain"
		source: "foo bar"
	}
	scratchConfig: oras.#repoBlob & {
		desc: oras.scratchConfig
		source: {}
	}

	// This is a blob that has the media type and contents of a normal manifest.
	blobManifest: oras.#repoBlob & {
		desc: mediaType: source.mediaType
		source: oras.#manifest & {
			schemaVersion: _
			mediaType: _
			artifactType: "application/dubious-manifest"
			config: scratchConfig.desc
			layers: [
				// Because this reference is inside a blob, not a manifest
				// (despite the manifest media type of the blob),
				// this shouldn't count as a hard reference.
				unreferencedBlob.desc,
			]
		}
	}
	actualManifest: oras.#repoManifest & {
		manifest:  oras.#manifest & {
			mediaType: _
			artifactType: "ok-manifest"
			config: oras.scratchConfig
			layers: [
				blobManifest.desc
			]
		}
	}
	tag: oras.#repoTag & {
		name: "test"
		desc: actualManifest.desc
	}
	dump: oras.#repoDump & {
		repo: _
		script: """
		oras blob delete -f \(host)/\(repo)@\(unreferencedBlob.desc.digest)
		oras copy -r \(host)/\(repo):\(tag.name) \(host)/\(repo)-copy:\(tag.name)
		"""
	}
}
