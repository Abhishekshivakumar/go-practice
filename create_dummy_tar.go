package main

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DummyManifest represents the manifest.json file in a Docker image archive.
type DummyManifest []struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

// generateSHA256Hash generates a fake SHA256 hash to simulate Docker layer names.
func generateSHA256Hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// createDummyLayer creates a simulated Docker layer with required metadata.
func createDummyLayer(layerDir string) error {
	// Create layer.tar
	layerTarPath := filepath.Join(layerDir, "layer.tar")
	layerFile, err := os.Create(layerTarPath)
	if err != nil {
		return fmt.Errorf("error creating layer tar file: %v", err)
	}
	defer layerFile.Close()

	tarWriter := tar.NewWriter(layerFile)
	defer tarWriter.Close()

	content := []byte("Hello, this is a Docker layer!")
	header := &tar.Header{
		Name:    "dummy.txt",
		Mode:    0600,
		Size:    int64(len(content)),
		ModTime: time.Now(),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("error writing tar header: %v", err)
	}

	if _, err := tarWriter.Write(content); err != nil {
		return fmt.Errorf("error writing tar content: %v", err)
	}

	// Create VERSION file
	err = os.WriteFile(filepath.Join(layerDir, "VERSION"), []byte("1.0"), 0644)
	if err != nil {
		return fmt.Errorf("error writing VERSION file: %v", err)
	}

	// Create json metadata
	layerMetadata := `{"id": "` + filepath.Base(layerDir) + `"}`
	err = os.WriteFile(filepath.Join(layerDir, "json"), []byte(layerMetadata), 0644)
	if err != nil {
		return fmt.Errorf("error writing json file: %v", err)
	}

	return nil
}

// createDockerArchive generates a Docker-like archive.
func createDockerArchive(outputFile string) error {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "docker-image")
	if err != nil {
		return fmt.Errorf("error creating temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Cleanup after execution

	// Generate layer directories with SHA256 names
	layerHashes := []string{
		generateSHA256Hash("layer1"),
		generateSHA256Hash("layer2"),
	}
	var layers []string

	for _, layerHash := range layerHashes {
		layerDir := filepath.Join(tempDir, layerHash)
		os.Mkdir(layerDir, 0755)

		err = createDummyLayer(layerDir)
		if err != nil {
			return err
		}
		layers = append(layers, layerHash+"/layer.tar")
	}

	// Create config.json with SHA256 hash as filename
	configData := map[string]interface{}{
		"architecture": "amd64",
		"os":           "linux",
		"config": map[string]interface{}{
			"Env": []string{"Dummy=1"},
		},
	}
	configBytes, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("error creating config.json: %v", err)
	}

	configHash := generateSHA256Hash("config")
	configFileName := configHash + ".json"
	err = os.WriteFile(filepath.Join(tempDir, configFileName), configBytes, 0644)
	if err != nil {
		return fmt.Errorf("error writing config.json: %v", err)
	}

	// Create manifest.json
	manifest := DummyManifest{
		{
			Config:   configFileName,
			RepoTags: []string{"dummyimage:latest"},
			Layers:   layers,
		},
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("error creating manifest.json: %v", err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "manifest.json"), manifestBytes, 0644)
	if err != nil {
		return fmt.Errorf("error writing manifest.json: %v", err)
	}

	// Create final tar archive
	outputFileHandle, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer outputFileHandle.Close()

	tarWriter := tar.NewWriter(outputFileHandle)
	defer tarWriter.Close()

	// Add files to archive
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		header := &tar.Header{
			Name:    relPath,
			Mode:    int64(info.Mode()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		_, err = file.WriteTo(tarWriter)
		return err
	})

	if err != nil {
		return fmt.Errorf("error writing tar archive: %v", err)
	}

	fmt.Println("Docker archive created successfully:", outputFile)
	return nil
}

// main function
func CreateTar(fileName string) {
	// outputFile := "dummy-docker-image.tar"
	err := createDockerArchive(fileName)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Dummy Docker image archive created:", fileName)
	}
}
