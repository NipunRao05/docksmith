package cli

import (
	"docksmith/internal/model"
	"docksmith/internal/storage"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func HandleImages() error {
	images, err := storage.ListImages()
	if err != nil {
		return err
	}
	if len(images) == 0 {
		fmt.Println("No images found")
		return nil
	}
	fmt.Printf("%-20s %-10s %-15s %-25s\n", "REPOSITORY", "TAG", "IMAGE ID", "CREATED")
	for _, file := range images {
		path := filepath.Join(storage.ImagesDir(), file)
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		var img model.Image
		err = json.NewDecoder(f).Decode(&img)
		f.Close()
		if err != nil {
			return err
		}
		digest := img.Digest
		if strings.HasPrefix(digest, "sha256:") {
			digest = digest[7:]
		}
		if len(digest) > 12 {
			digest = digest[:12]
		}
		fmt.Printf("%-20s %-10s %-15s %-25s\n", img.Name, img.Tag, digest, img.Created)
	}
	return nil
}
