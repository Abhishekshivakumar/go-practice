package main

import (
	"fmt"
)

const (
	SourceUnknown ImageSource = iota
	SourceDockerArchive
)

type ImageSource int

func (r ImageSource) String() string {
	return [...]string{"unknown", "docker-archive"}[r]
}

var ImageSources = []string{SourceUnknown.String(), SourceDockerArchive.String()}

func GetImageResolver(r ImageSource) (Resolver, error) {
	switch r {
	case SourceDockerArchive:
		return NewResolverFromArchive(), nil
	}

	return nil, fmt.Errorf("unable to determine image resolver")
}
