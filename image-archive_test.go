package main

import (
	"archive/tar"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewImageArchive(t *testing.T) {
	// Create a dummy tar archive for testing
	tarData := createTestTarArchive(t)
	tarReader := io.NopCloser(bytes.NewReader(tarData))

	img, err := NewImageArchive(tarReader)
	require.NoError(t, err, "NewImageArchive failed")

	// Add assertions based on your expected image structure
	require.NotNil(t, img, "Image archive is nil")
	require.Equal(t, "config.json", img.manifest.ConfigPath, "Expected config path 'config.json'")
	require.Len(t, img.layerMap, 2, "Expected 2 layers")
}

func TestProcessLayer(t *testing.T) {
	img := &ImageArchive{layerMap: make(map[string]*FileTree)}
	tarData := createLayerTarArchive(t)
	tarReader := tar.NewReader(bytes.NewReader(tarData))
	header, _ := tarReader.Next() // Get first header

	err := processLayer(img, tarReader, header.Name)
	require.NoError(t, err, "processLayer failed")
	require.Len(t, img.layerMap, 1, "Expected 1 layer")
}

func TestProcessBlob(t *testing.T) {
	img := &ImageArchive{layerMap: make(map[string]*FileTree)}
	jsonFiles := make(map[string][]byte)
	tarData := createBlobTarArchive(t)
	tarReader := tar.NewReader(bytes.NewReader(tarData))
	header, _ := tarReader.Next()

	err := processBlob(img, tarReader, header.Name, jsonFiles)
	require.NoError(t, err, "processBlob failed")
	require.True(t, len(img.layerMap) == 1 || len(jsonFiles) == 1, "Expected 1 layer or json file")
}

func TestToImage(t *testing.T) {
	img := &ImageArchive{
		manifest: manifest{LayerTarPaths: []string{"layer1.tar", "layer2.tar"}, ConfigPath: "config.json"},
		config:   config{History: []historyEntry{{CreatedBy: "test", EmptyLayer: false}, {CreatedBy: "test2", EmptyLayer: false}}},
		layerMap: map[string]*FileTree{
			"layer1.tar": {Name: "layer1.tar", FileSize: 100},
			"layer2.tar": {Name: "layer2.tar", FileSize: 200},
		},
	}

	image, err := img.ToImage()
	require.NoError(t, err, "ToImage failed")
	require.Len(t, image.Layers, 2, "Expected 2 layers in image")
	require.Equal(t, int64(100), image.Layers[0].Size, "Expected layer size 100")
	require.Equal(t, int64(200), image.Layers[1].Size, "Expected layer size 200")
}

func createTestTarArchive(t *testing.T) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Add manifest.json
	addFileToTar(t, tw, "manifest.json", `{"ConfigPath":"config.json", "LayerTarPaths":["layer1.tar", "layer2.tar"]}`)
	// Add config.json
	addFileToTar(t, tw, "config.json", `{"History":[{"CreatedBy":"test", "EmptyLayer":false}, {"CreatedBy":"test2", "EmptyLayer":false}]}`)
	// Add layer1.tar
	addLayerTar(t, tw, "layer1.tar", "file1.txt", "content1")
	// Add layer2.tar
	addLayerTar(t, tw, "layer2.tar", "file2.txt", "content2")

	tw.Close()
	return buf.Bytes()
}

func createLayerTarArchive(t *testing.T) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	addFileToTar(t, tw, "file.txt", "content")
	tw.Close()
	return buf.Bytes()
}

func createBlobTarArchive(t *testing.T) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	addFileToTar(t, tw, "blobs/config.json", `{"history":[]}`)
	addLayerTar(t, tw, "blobs/layer.tar", "file.txt", "content")
	tw.Close()
	return buf.Bytes()
}

func addFileToTar(t *testing.T, tw *tar.Writer, name, content string) {
	hdr := &tar.Header{
		Name: name,
		Mode: 0600,
		Size: int64(len(content)),
	}
	require.NoError(t, tw.WriteHeader(hdr), "WriteHeader failed")
	_, err := tw.Write([]byte(content))
	require.NoError(t, err, "Write failed")
}

func addLayerTar(t *testing.T, tw *tar.Writer, layerName, fileName, content string) {
	var layerBuf bytes.Buffer
	layerTw := tar.NewWriter(&layerBuf)
	addFileToTar(t, layerTw, fileName, content)
	layerTw.Close()

	hdr := &tar.Header{
		Name: layerName,
		Mode: 0600,
		Size: int64(layerBuf.Len()),
	}
	require.NoError(t, tw.WriteHeader(hdr), "WriteHeader failed")
	_, err := tw.Write(layerBuf.Bytes())
	require.NoError(t, err, "Write failed")
}
