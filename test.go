package main

import (
	"os"
)

type podmanResolver struct{}

func NewResolverFromPodman() *podmanResolver {
	return &podmanResolver{}
}

func (r *archiveResolver) Fetch(path string) (*Image, error) {
	return r.Build(path)
}

func (r *archiveResolver) Build(arg string) (*Image, error) {
	archivePath, err := someFunctionToPerfromPodmanCommand(arg)
	reader, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	img, err := newImageArchiveFunc(reader)
	if err != nil {
		return nil, err
	}
	return img.ToImage()
}
