package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type scriptRegistry struct {
	host   string
	fileID int32
}

func newScriptRegistry(host string) *scriptRegistry {
	fmt.Printf("#!/bin/sh\n")
	fmt.Printf("set -ex\n")
	fmt.Printf("tmpRoot=$(mktemp -d)\n")
	return &scriptRegistry{
		host: host,
	}
}

func (r *scriptRegistry) Push(ctx context.Context, repoName string, desc ocispec.Descriptor, content io.Reader) error {
	f, err := r.writeFile(content, desc.MediaType)
	if err != nil {
		return err
	}
	fmt.Printf("oras blob push --media-type %s %s/%s@%s %s\n", desc.MediaType, r.host, repoName, desc.Digest, f)
	return nil
}

func (r *scriptRegistry) PushManifest(ctx context.Context, repoName string, desc ocispec.Descriptor, content io.Reader) error {
	f, err := r.writeFile(content, desc.MediaType)
	if err != nil {
		return err
	}
	fmt.Printf("oras manifest push --media-type %s %s/%s@%s %s\n", desc.MediaType, r.host, repoName, desc.Digest, f)
	return nil
}

func (r *scriptRegistry) Dump(ctx context.Context, stuff json.RawMessage) {
	var p struct {
		Script string `json:"script"`
	}
	if err := json.Unmarshal(stuff, &p); err != nil {
		fmt.Printf("# bad script %q\n", stuff)
		return
	}
	fmt.Printf("%s\n", p.Script)
}

func shQuote(s string) string {
	return `'` + strings.Replace(s, `'`, `'"'"'`, -1) + `'`
}

func (r *scriptRegistry) Tag(ctx context.Context, repoName string, desc ocispec.Descriptor, reference string) error {
	fmt.Printf("oras tag %s/%s@%s %s\n", r.host, repoName, desc.Digest, reference)
	return nil
}

func (r *scriptRegistry) writeFile(content io.Reader, mediaType string) (string, error) {
	data, err := ioutil.ReadAll(content)
	if err != nil {
		return "", err
	}
	f := r.tmpFile()
	if utf8.Valid(data) {
		fmt.Printf("> %s echo -n %s\n", f, shQuote(string(data)))
		return f, nil
	}
	s := base64.StdEncoding.EncodeToString(data)
	fmt.Printf("echo %s | base64 -d > %s\n", s, f)
	return f, nil
}

func (r *scriptRegistry) tmpFile() string {
	id := atomic.AddInt32(&r.fileID, 1)
	return fmt.Sprintf("$tmpRoot/%d", id)
}
