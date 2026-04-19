package cli

import (
	"compress/gzip"
	"docksmith/internal/model"
	"docksmith/internal/storage"
	"docksmith/internal/utils"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func HandleImport(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: docksmith import <tar-file> <name:tag>")
	}

	tarPath := args[0]
	imageRef := args[1]

	// Validate tar file exists
	if _, err := os.Stat(tarPath); err != nil {
		return fmt.Errorf("tar file not found: %s", tarPath)
	}

	// Decompress if gzipped
	finalTarPath := tarPath
	if strings.HasSuffix(tarPath, ".gz") {
		decompressed, err := decompressGzip(tarPath)
		if err != nil {
			return fmt.Errorf("failed to decompress: %v", err)
		}
		defer os.Remove(decompressed)
		finalTarPath = decompressed
	}

	// Parse image reference (name:tag)
	parts := strings.Split(imageRef, ":")
	var name, tag string
	if len(parts) == 2 {
		name = parts[0]
		tag = parts[1]
	} else if len(parts) == 1 {
		name = parts[0]
		tag = "latest"
	} else {
		return fmt.Errorf("invalid image reference: %s (expected name or name:tag)", imageRef)
	}

	// Compute layer digest from tar file
	digest, err := utils.HashFile(finalTarPath)
	if err != nil {
		return fmt.Errorf("failed to compute digest: %v", err)
	}

	// Save the layer
	layerPath, err := storage.SaveLayer(finalTarPath, digest)
	if err != nil {
		return fmt.Errorf("failed to save layer: %v", err)
	}

	// Get layer size
	info, err := os.Stat(layerPath)
	if err != nil {
		return fmt.Errorf("failed to stat layer: %v", err)
	}

	// Create image manifest
	img := model.Image{
		Name:    name,
		Tag:     tag,
		Digest:  digest,
		Created: time.Now().UTC().Format(time.RFC3339),
		Config: model.Config{
			Env:        []string{},
			Cmd:        []string{},
			WorkingDir: "/",
		},
		Layers: []model.Layer{
			{
				Digest:    digest,
				Size:      info.Size(),
				CreatedBy: "docksmith import",
			},
		},
	}

	// Save image manifest
	err = storage.SaveImage(img)
	if err != nil {
		return fmt.Errorf("failed to save image manifest: %v", err)
	}

	fmt.Printf("Imported %s:%s with digest %s\n", name, tag, digest[:19])
	return nil
}

func decompressGzip(gzPath string) (string, error) {
	gzFile, err := os.Open(gzPath)
	if err != nil {
		return "", err
	}
	defer gzFile.Close()

	gzReader, err := gzip.NewReader(gzFile)
	if err != nil {
		return "", err
	}
	defer gzReader.Close()

	tmpFile, err := os.CreateTemp("", "docksmith-import-*.tar")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, gzReader); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}
