package builder

import (
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
		layers = append(layers, model.Layer{
			Digest:    digest,
			Size:      size,
			CreatedBy: "",
		})
	}

	if len(state.Layers) == 0 {
		return fmt.Errorf("no layers produced")
	}

	img := model.Image{
		Name:    name,
		Tag:     tagName,
		Digest:  state.Layers[len(state.Layers)-1],
		Created: time.Now().Format(time.RFC3339),
		Config: model.Config{
			Env:        envList,
			Cmd:        state.Cmd,
			WorkingDir: state.WorkingDir,
		},
		Layers: layers,
	}

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
