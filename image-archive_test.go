package main

import (
	"archive/tar"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewImageArchive_Success(t *testing.T) {
	// Create a dummy tar archive for testing
	tarData := createTestTarArchive(t)
	tarReader := io.NopCloser(bytes.NewReader(tarData))

	img, err := NewImageArchive(tarReader)
	require.NoError(t, err, "NewImageArchive failed")
	require.NotNil(t, img, "Image archive is nil")
	require.Equal(t, "config.json", img.manifest.ConfigPath, "Expected config path 'config.json'")
	require.Len(t, img.layerMap, 2, "Expected 2 layers")
}

func TestNewImageArchive_InvalidTar(t *testing.T) {
	// Create an invalid tar archive (e.g., just random bytes)
	tarData := []byte{0x01, 0x02, 0x03}
	tarReader := io.NopCloser(bytes.NewReader(tarData))

	img, err := NewImageArchive(tarReader)
	require.Error(t, err, "Expected error for invalid tar")
	require.Nil(t, img, "Expected nil image for invalid tar")
}

func TestNewImageArchive_MissingManifest(t *testing.T) {
	// Create a tar archive without manifest.json
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	addFileToTar(t, tw, "config.json", `{"History":[]}`) // Config, but no manifest
	addLayerTar(t, tw, "layer1.tar", "file1.txt", "content1")
	tw.Close()

	tarReader := io.NopCloser(bytes.NewReader(buf.Bytes()))

	img, err := NewImageArchive(tarReader)
	require.Error(t, err, "Expected error for missing manifest")
	require.Nil(t, img, "Expected nil image for missing manifest")
	require.ErrorIs(t, err, ErrManifestNotFound, "Expected ErrManifestNotFound")
}

func TestNewImageArchive_MissingConfig(t *testing.T) {
	// Create a tar archive with manifest.json but without config.json
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	addFileToTar(t, tw, "manifest.json", `{"ConfigPath":"config.json", "LayerTarPaths":["layer1.tar"]}`) // Manifest, but no config
	addLayerTar(t, tw, "layer1.tar", "file1.txt", "content1")
	tw.Close()

	tarReader := io.NopCloser(bytes.NewReader(buf.Bytes()))

	img, err := NewImageArchive(tarReader)
	require.Error(t, err, "Expected error for missing config")
	require.Nil(t, img, "Expected nil image for missing config")
	require.ErrorIs(t, err, ErrConfigNotFound, "Expected ErrConfigNotFound")
}

// Helper functions (createTestTarArchive, addFileToTar, addLayerTar) from previous examples...
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
