module github.com/cue-exp/oras

go 1.21

require (
	cuelang.org/go v0.6.0-0.dev.0.20230328131919-be0601bf379c
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0-rc2
	golang.org/x/mod v0.9.0
	oras.land/oras-go/v2 v2.0.2
)

require (
	github.com/cockroachdb/apd/v2 v2.0.2 // indirect
	github.com/emicklei/proto v1.10.0 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mpvl/unique v0.0.0-20150818121801-cbe035fff7de // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/protocolbuffers/txtpbfmt v0.0.0-20230328191034-3462fbc510c0 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace cuelang.org/go => /home/rogpeppe/src/cuelabs/cue

replace oras.land/oras-go/v2 => /home/rogpeppe/gohack/oras.land/oras-go/v2
