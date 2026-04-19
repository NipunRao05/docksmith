package cli

import (
	"docksmith/internal/storage"
	"errors"
	"fmt"
	"strings"
)

func HandleRMI(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: docksmith rmi <name:tag>")
	}
	nameTag := args[0]
	filename := strings.ReplaceAll(nameTag, ":", "_") + ".json"

	// Load image first to get layer digests
	img, err := storage.LoadImage(filename)
	if err != nil {
		return fmt.Errorf("image not found: %s", nameTag)
	}

	// Delete each layer file
	for _, layer := range img.Layers {
		layerFile := strings.ReplaceAll(layer.Digest, ":", "_") + ".tar"
		err := storage.RemoveLayer(layerFile)
		if err != nil {
			fmt.Printf("warning: could not remove layer %s: %v\n", layer.Digest, err)
		} else {
			fmt.Printf("Deleted layer: %s\n", layer.Digest)
		}
	}

	// Delete image manifest
	err = storage.RemoveImage(filename)
	if err != nil {
		return err
	}
	fmt.Printf("Deleted image: %s\n", nameTag)
	return nil
}
