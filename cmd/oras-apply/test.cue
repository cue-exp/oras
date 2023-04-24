package test

//import "github.com/cue-exp/oras"
//
//blobs: [_]: oras.#repoBlob
//
//manifests: [_]: oras.#repoManifest
//
//tags: [name=_]: oras.#repoTag & {
//	"name": name
//}
//
//_module: "github.com/some/module"
//
//blobs: {
//	foo: {
//		desc: mediaType: "text/plain"
//		repo:   _module
//		source: "foo bar"
//	}
//	someContent: {
//		desc: mediaType: "application/zip"
//		repo:   _module
//		source: """
//		-- cue.mod/module.cue --
//		module: "\(_module)"
//
//		something: "\(foo.desc.digest)"
//		-- foo.cue --
//		x: "hello world"
//		"""
//	}
//}
//
//manifests: bar: {
//	repo: _module
//	manifest: {
//		schemaVersion: _
//		config:        blobs.foo.desc
//		layers: [
//			blobs.someContent.desc,
//		]
//	}
//}
//
//tags: "v1.1.0": {
//	repo:   _module
//	digest: manifests.bar.desc.digest
//}
