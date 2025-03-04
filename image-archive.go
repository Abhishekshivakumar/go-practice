package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

type ImageArchive struct {
	manifest manifest
	config   config
	layerMap map[string]*FileTree
}

var (
	ErrManifestNotFound = errors.New("could not find image manifest")
	ErrConfigNotFound   = errors.New("could not find image config")
	ErrLayerParseFailed = errors.New("failed to parse layer tar")
)

const (
	tarExtension   = ".tar"
	tarGzExtension = ".tar.gz"
	tgzExtension   = ".tgz"
	manifestFile   = "manifest.json"
	blobsPrefix    = "blobs/"
	bufferSize     = 1024
)

func NewImageArchive(tarFile io.ReadCloser) (*ImageArchive, error) {
	img := &ImageArchive{
		layerMap: make(map[string]*FileTree),
	}

	tarReader := tar.NewReader(tarFile)
	jsonFiles := make(map[string][]byte)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading tar: %w", err)
		}

		if err := processTarEntry(img, tarReader, header, jsonFiles); err != nil {
			return nil, err
		}
	}

	if err := img.loadManifestAndConfig(jsonFiles); err != nil {
		return nil, err
	}

	return img, nil
}

func processTarEntry(img *ImageArchive, tarReader *tar.Reader, header *tar.Header, jsonFiles map[string][]byte) error {
	name := header.Name

	if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeReg {
		ext := path.Ext(name)
		if ext == tarExtension || ext == tarGzExtension || ext == tgzExtension {
			return processLayer(img, tarReader, name)
		} else if path.Ext(name) == ".json" || strings.HasPrefix(name, "sha256:") {
			return processJSON(tarReader, name, jsonFiles)
		} else if strings.HasPrefix(name, blobsPrefix) {
			return processBlob(img, tarReader, name, jsonFiles)
		}
	}
	return nil
}

func processLayer(img *ImageArchive, tarReader *tar.Reader, name string) error {
	var layerReader *tar.Reader
	var err error

	if path.Ext(name) == tarGzExtension || path.Ext(name) == tgzExtension {
		gz, err := gzip.NewReader(tarReader)
		if err != nil {
			return err
		}
		layerReader = tar.NewReader(gz)
	} else {
		layerReader = tar.NewReader(tarReader)
	}

	tree, err := processLayerTar(name, layerReader)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrLayerParseFailed, name)
	}

	img.layerMap[tree.Name] = tree
	return nil
}

func processJSON(tarReader *tar.Reader, name string, jsonFiles map[string][]byte) error {
	fileBuffer, err := io.ReadAll(tarReader)
	if err != nil {
		return err
	}
	jsonFiles[name] = fileBuffer
	return nil
}

func processBlob(img *ImageArchive, tarReader *tar.Reader, name string, jsonFiles map[string][]byte) error {
	buffer := make([]byte, bufferSize)
	n, err := io.ReadFull(tarReader, buffer)
	if err != nil && err != io.ErrUnexpectedEOF {
		return err
	}

	if n == bufferSize {
		if err := tryProcessTarBlob(img, bytes.NewReader(buffer[:n]), tarReader, name); err == nil {
			return nil
		}
	}

	return tryProcessJSONBlob(bytes.NewReader(buffer[:n]), tarReader, name, jsonFiles)
}

func tryProcessTarBlob(img *ImageArchive, bufferReader io.Reader, tarReader io.Reader, name string) error {
	multiReader := io.MultiReader(bufferReader, tarReader)
	gzipReader, err := gzip.NewReader(multiReader)
	if err != nil {
		// Not a gzipped entry, use the MultiReader directly
		layerReader := tar.NewReader(multiReader)
		tree, err := processLayerTar(name, layerReader)
		if err != nil {
			return err
		}

		img.layerMap[tree.Name] = tree
		return nil
	}

	// Gzip decompression successful, use the gzipReader
	layerReader := tar.NewReader(gzipReader)
	tree, err := processLayerTar(name, layerReader)
	if err != nil {
		return err
	}

	img.layerMap[tree.Name] = tree
	return nil
}

func tryProcessJSONBlob(bufferReader io.Reader, tarReader io.Reader, name string, jsonFiles map[string][]byte) error {
	decoder := json.NewDecoder(bufferReader)
	token, err := decoder.Token()
	if _, ok := token.(json.Delim); err == nil && ok {
		fileBuffer, err := io.ReadAll(io.MultiReader(bufferReader, tarReader))
		if err != nil {
			return err
		}
		jsonFiles[name] = fileBuffer
		return nil
	}
	return nil
}

func (img *ImageArchive) loadManifestAndConfig(jsonFiles map[string][]byte) error {
	manifestContent, exists := jsonFiles[manifestFile]
	if !exists {
		return ErrManifestNotFound
	}

	img.manifest = newManifest(manifestContent)

	configContent, exists := jsonFiles[img.manifest.ConfigPath]
	if !exists {
		return ErrConfigNotFound
	}

	img.config = newConfig(configContent)
	return nil
}

func processLayerTar(name string, reader *tar.Reader) (*FileTree, error) {
	tree := NewFileTree()
	tree.Name = name

	fileInfos, err := getFileList(reader)
	if err != nil {
		return nil, err
	}

	for _, element := range fileInfos {
		tree.FileSize += uint64(element.Size)

		_, _, err := tree.AddPath(element.Path, element)
		if err != nil {
			return nil, err
		}
	}

	return tree, nil
}

func getFileList(tarReader *tar.Reader) ([]FileInfo, error) {
	var files []FileInfo

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		name := path.Clean(header.Name)
		if name == "." {
			continue
		}

		switch header.Typeflag {
		case tar.TypeXGlobalHeader:
			return nil, fmt.Errorf("unexptected tar file: (XGlobalHeader): type=%v name=%s", header.Typeflag, name)
		case tar.TypeXHeader:
			return nil, fmt.Errorf("unexptected tar file (XHeader): type=%v name=%s", header.Typeflag, name)
		default:
			files = append(files, NewFileInfoFromTarHeader(tarReader, header, name))
		}
	}
	return files, nil
}

type layer struct {
	history historyEntry
	index   int
	tree    *FileTree
}

func (l *layer) ToLayer() *Layer {
	id := strings.Split(l.tree.Name, "/")[0]
	return &Layer{
		Id:      id,
		Index:   l.index,
		Command: strings.TrimPrefix(l.history.CreatedBy, "/bin/sh -c "),
		Size:    l.history.Size,
		Tree:    l.tree,
		Names:   []string{"(unavailable)"},
		Digest:  l.history.ID,
	}
}

func (img *ImageArchive) ToImage() (*Image, error) {
	trees := make([]*FileTree, 0)

	// build the content tree
	for _, treeName := range img.manifest.LayerTarPaths {
		tr, exists := img.layerMap[treeName]
		if exists {
			trees = append(trees, tr)
			continue
		}
		return nil, fmt.Errorf("could not find '%s' in parsed layers", treeName)
	}

	// build the layers array
	layers := make([]*Layer, 0)

	// note that the engineResolver config stores images in reverse chronological order, so iterate backwards through layers
	// as you iterate chronologically through history (ignoring history items that have no layer contents)
	// Note: history is not required metadata in a docker image!
	histIdx := 0
	for idx, tree := range trees {
		// ignore empty layers, we are only observing layers with content
		historyObj := historyEntry{
			CreatedBy: "(missing)",
		}
		for nextHistIdx := histIdx; nextHistIdx < len(img.config.History); nextHistIdx++ {
			if !img.config.History[nextHistIdx].EmptyLayer {
				histIdx = nextHistIdx
				break
			}
		}
		if histIdx < len(img.config.History) && !img.config.History[histIdx].EmptyLayer {
			historyObj = img.config.History[histIdx]
			histIdx++
		}

		historyObj.Size = tree.FileSize

		dockerLayer := layer{
			history: historyObj,
			index:   idx,
			tree:    tree,
		}
		layers = append(layers, dockerLayer.ToLayer())
	}

	return &Image{
		Trees:  trees,
		Layers: layers,
	}, nil
}
