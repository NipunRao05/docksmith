package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"docksmith/internal/model"
	"docksmith/internal/storage"
)

type Engine struct {
	NoCache bool
}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Build(tag string, context string) error {
	instructions, err := ParseDocksmithfile("Docksmithfile")
	if err != nil {
		return err
	}

	if err := validateBuildInstructions(instructions); err != nil {
		return err
	}

	total := len(instructions)
	buildStart := time.Now()

	state, err := ExecuteInstructionsWithOutput(instructions, e.NoCache, total)
	if err != nil {
		return err
	}

	parts := strings.Split(tag, ":")
	name := parts[0]
	tagName := "latest"
	if len(parts) > 1 {
		tagName = parts[1]
	}

	var envList []string
	for k, v := range state.Env {
		envList = append(envList, k+"="+v)
	}

	var layers []model.Layer
	for _, digest := range state.Layers {
		// Get size of layer file
		layerFile := strings.ReplaceAll(digest, ":", "_") + ".tar"
		size := storage.LayerSize(layerFile)
		createdBy := ""
		if cb, exists := state.LayerCreatedBy[digest]; exists {
			createdBy = cb
		}
		layers = append(layers, model.Layer{
			Digest:    digest,
			Size:      size,
			CreatedBy: createdBy,
		})
	}

	if len(state.Layers) == 0 {
		return fmt.Errorf("no layers produced")
	}

	// Check if image already exists to preserve timestamp on cache hits
	createdTime := time.Now().Format(time.RFC3339)
	existingManifestFile := name + "_" + tagName + ".json"
	if existingImg, err := storage.LoadImage(existingManifestFile); err == nil {
		// Image exists, preserve its creation timestamp
		createdTime = existingImg.Created
	}

	// Create image with empty digest for canonical JSON hashing
	img := model.Image{
		Name:    name,
		Tag:     tagName,
		Digest:  "", // Empty for hashing
		Created: createdTime,
		Config: model.Config{
			Env:        envList,
			Cmd:        state.Cmd,
			WorkingDir: state.WorkingDir,
		},
		Layers: layers,
	}

	// Compute manifest digest from canonical JSON
	jsonBytes, err := json.Marshal(img)
	if err != nil {
		return fmt.Errorf("failed to marshal image: %v", err)
	}
	hash := sha256.Sum256(jsonBytes)
	img.Digest = "sha256:" + hex.EncodeToString(hash[:])

	err = storage.SaveImage(img)
	if err != nil {
		return err
	}

	totalTime := time.Since(buildStart)
	shortDigest := img.Digest
	if strings.HasPrefix(shortDigest, "sha256:") {
		shortDigest = shortDigest[7:15]
	}
	fmt.Printf("Successfully built sha256:%s %s (%.2fs)\n", shortDigest, tag, totalTime.Seconds())
	return nil
}

func validateBuildInstructions(instructions []model.Instruction) error {
	if len(instructions) == 0 {
		return fmt.Errorf("Docksmithfile is empty")
	}

	fromIndex := -1
	for i, inst := range instructions {
		if inst.Type != "FROM" {
			continue
		}

		if fromIndex != -1 {
			return fmt.Errorf("multiple FROM instructions are not supported (line %d)", inst.Line)
		}

		fromIndex = i
	}

	if fromIndex == -1 {
		return fmt.Errorf("missing FROM instruction")
	}

	if fromIndex != 0 {
		return fmt.Errorf("FROM must be the first instruction (line %d)", instructions[fromIndex].Line)
	}

	return nil
}
