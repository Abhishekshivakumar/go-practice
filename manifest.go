package main

import (
	"encoding/json"
	"fmt"
)

type manifest struct {
	ConfigPath    string   `json:"Config"`
	RepoTags      []string `json:"RepoTags"`
	LayerTarPaths []string `json:"Layers"`
}

func newManifest(manifestBytes []byte) manifest {
	var manifest []manifest
	err := json.Unmarshal(manifestBytes, &manifest)
	if err != nil {
		fmt.Println("Error: while creating json  ", err)
	}
	return manifest[0]
}
