package builder

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"docksmith/internal/model"
	"docksmith/internal/runtime"
	"docksmith/internal/storage"
	"docksmith/internal/utils"
)

func ExecuteInstructionsWithOutput(instructions []model.Instruction, noCache bool, total int) (*BuildState, error) {
	cache, err := storage.LoadCache()
	if err != nil {
		return nil, err
	}

	cacheUpdated := false
	cacheMissed := false

	root, err := os.MkdirTemp("", "docksmith-build-")
	if err != nil {
		return nil, err
	}

	state := &BuildState{
		WorkingDir:             "/",
		Env:                    make(map[string]string),
		RootFS:                 root,
		LayerCreatedBy:         make(map[string]string),
		PreviousFileHashes:     make(map[string]string),
	}

	for i, inst := range instructions {
		stepNum := i + 1

		switch inst.Type {

		case "FROM":
			fmt.Printf("Step %d/%d : FROM %s\n", stepNum, total, strings.Join(inst.Args, " "))
			if err := handleFrom(inst, state); err != nil {
				return nil, err
			}

		case "WORKDIR":
			fmt.Printf("Step %d/%d : WORKDIR %s\n", stepNum, total, strings.Join(inst.Args, " "))
			state.WorkingDir = inst.Args[0]
			absPath := filepath.Join(state.RootFS, state.WorkingDir)
			if err := os.MkdirAll(absPath, 0755); err != nil {
				return nil, err
			}

		case "ENV":
			fmt.Printf("Step %d/%d : ENV %s\n", stepNum, total, strings.Join(inst.Args, " "))
			handleEnv(inst, state)

		case "CMD":
			fmt.Printf("Step %d/%d : CMD %s\n", stepNum, total, strings.Join(inst.Args, " "))
			state.Cmd = inst.Args

		case "COPY":
			key := generateCacheKey(inst, state)
			stepStart := time.Now()

			if !noCache && !cacheMissed {
				if digest, ok := cache[key]; ok {
					layerFile := strings.ReplaceAll(digest, ":", "_") + ".tar"
					layerPath := filepath.Join(storage.LayersDir(), layerFile)
					if _, err := os.Stat(layerPath); err == nil {
						if err := utils.ExtractTar(layerPath, state.RootFS); err != nil {
							return nil, fmt.Errorf("failed to apply cached COPY layer %s: %w", digest, err)
						}
						fmt.Printf("Step %d/%d : COPY %s [CACHE HIT]\n", stepNum, total, strings.Join(inst.Args, " "))
						recordProducedLayer(state, digest)					state.LayerCreatedBy[digest] = inst.Raw						continue
					}
				}
			}

			// CACHE MISS
			if err := handleCopy(inst, state); err != nil {
				return nil, err
			}

			elapsed := time.Since(stepStart)
			fmt.Printf("Step %d/%d : COPY %s [CACHE MISS] %.2fs\n", stepNum, total, strings.Join(inst.Args, " "), elapsed.Seconds())

			cacheMissed = true

			if !noCache {
				cache[key] = state.Layers[len(state.Layers)-1]
				cacheUpdated = true
			}

		case "RUN":
			key := generateCacheKey(inst, state)
			stepStart := time.Now()

			if !noCache && !cacheMissed {
				if digest, ok := cache[key]; ok {
					layerFile := strings.ReplaceAll(digest, ":", "_") + ".tar"
					layerPath := filepath.Join(storage.LayersDir(), layerFile)
					if _, err := os.Stat(layerPath); err == nil {
						if err := utils.ExtractTar(layerPath, state.RootFS); err != nil {
							return nil, fmt.Errorf("failed to apply cached RUN layer %s: %w", digest, err)
						}
						fmt.Printf("Step %d/%d : RUN %s [CACHE HIT]\n", stepNum, total, strings.Join(inst.Args, " "))
						recordProducedLayer(state, digest)					state.LayerCreatedBy[digest] = inst.Raw						continue
					}
				}
			}

			if err := handleRun(inst, state); err != nil {
				return nil, err
			}
			elapsed := time.Since(stepStart)
			fmt.Printf("Step %d/%d : RUN %s [CACHE MISS] %.2fs\n", stepNum, total, strings.Join(inst.Args, " "), elapsed.Seconds())
			cacheMissed = true

			if !noCache {
				cache[key] = state.Layers[len(state.Layers)-1]
				cacheUpdated = true
			}

		default:
			return nil, fmt.Errorf("unknown instruction: %s at line %d", inst.Type, inst.Line)
		}
	}

	if cacheUpdated {
		if err := storage.SaveCache(cache); err != nil {
			return nil, err
		}
	}
	return state, nil
}

func handleEnv(inst model.Instruction, state *BuildState) {
	for _, pair := range inst.Args {
		for i := 0; i < len(pair); i++ {
			if pair[i] == '=' {
				key := pair[:i]
				value := pair[i+1:]
				state.Env[key] = value
				//fmt.Println("Set ENV:", key, "=", value)
				break
			}
		}
	}
}

func handleFrom(inst model.Instruction, state *BuildState) error {
	if len(inst.Args) != 1 {
		return fmt.Errorf("FROM requires exactly one image reference at line %d", inst.Line)
	}

	name, tag, err := parseImageRef(inst.Args[0])
	if err != nil {
		return fmt.Errorf("invalid FROM reference at line %d: %w", inst.Line, err)
	}

	manifestFile := name + "_" + tag + ".json"
	img, err := storage.LoadImage(manifestFile)
	if err != nil {
		return fmt.Errorf("base image %s:%s not found in local store", name, tag)
	}

	state.BaseImageDigest = img.Digest
	state.LastProducedLayerDigest = ""
	state.ProducedLayerCount = 0

	for _, layer := range img.Layers {
		layerFile := strings.ReplaceAll(layer.Digest, ":", "_") + ".tar"
		layerPath := filepath.Join(storage.LayersDir(), layerFile)
		if _, err := os.Stat(layerPath); err != nil {
			return fmt.Errorf("base layer missing for %s:%s: %s", name, tag, layer.Digest)
		}

		if err := utils.ExtractTar(layerPath, state.RootFS); err != nil {
			return fmt.Errorf("failed to extract base layer %s: %w", layer.Digest, err)
		}

		state.Layers = append(state.Layers, layer.Digest)
	}

	// Compute initial file hashes after loading base image
	// This allows us to detect changes in subsequent COPY/RUN instructions
	if hashes, err := utils.ComputeFileHashes(state.RootFS); err == nil {
		state.PreviousFileHashes = hashes
	}

	return nil
}

func parseImageRef(ref string) (string, string, error) {
	if ref == "" {
		return "", "", fmt.Errorf("empty image reference")
	}

	parts := strings.SplitN(ref, ":", 2)
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return "", "", fmt.Errorf("missing image name")
	}

	tag := "latest"
	if len(parts) == 2 {
		tag = strings.TrimSpace(parts[1])
		if tag == "" {
			return "", "", fmt.Errorf("missing image tag")
		}
	}

	return name, tag, nil
}

func recordProducedLayer(state *BuildState, digest string) {
	state.Layers = append(state.Layers, digest)
	state.LastProducedLayerDigest = digest
	state.ProducedLayerCount++
}

func copyEssentials(root string) error {
	binaries := []string{
		"/bin/sh",
		"/bin/ls",
		"/bin/touch",
		"/bin/mkdir",
		"/bin/rm",
		"/usr/bin/env",
	}

	for _, bin := range binaries {
		if err := copyHostBinary(bin, root); err == nil {
			_ = copyLibDeps(bin, root)
		}
	}
	return nil
}

func copyHostBinary(src, root string) error {
	if _, err := os.Stat(src); err != nil {
		return nil
	}

	dest := filepath.Join(root, src)

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return os.Chmod(dest, 0755)
}

func copyLibDeps(binary, root string) error {
	out, err := exec.Command("ldd", binary).Output()
	if err != nil {
		return nil
	}

	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		for _, f := range fields {
			if strings.HasPrefix(f, "/") {
				_ = copyHostBinary(f, root)
			}
		}
	}
	return nil
}

func handleCopy(inst model.Instruction, state *BuildState) error {
	if len(inst.Args) < 2 {
		return fmt.Errorf("COPY requires src and dest")
	}

	src := inst.Args[0]
	dest := inst.Args[1]
	targetPath := filepath.Join(state.RootFS, dest)

	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("COPY source not found: %s", src)
	}

	if srcInfo.IsDir() {
		// COPY dir/ /dest/ — copy directory contents
		if err := copyDir(src, targetPath); err != nil {
			return err
		}
	} else {
		// COPY file.txt /dest/ or /dest/file.txt
		destInfo, err := os.Stat(targetPath)
		if err == nil && destInfo.IsDir() {
			// destination is a directory — copy file into it
			targetPath = filepath.Join(targetPath, filepath.Base(src))
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		if err := copyFile(src, targetPath); err != nil {
			return err
		}
	}

	return createLayerFromState(state, inst.Raw)
}
func copyDir(src string, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func handleRun(inst model.Instruction, state *BuildState) error {
	command := strings.Join(inst.Args, " ")

	env := []string{}
	for k, v := range state.Env {
		env = append(env, k+"="+v)
	}

	// Copy essentials to a temp overlay dir, run there, then only
	// capture the rootfs changes (not the injected binaries)
	if err := runtime.RunIsolated(state.RootFS, state.WorkingDir, []string{command}, env); err != nil {
		return fmt.Errorf("RUN failed: %v", err)
	}
	if err := copyEssentials(state.RootFS); err != nil {
		return fmt.Errorf("failed to setup essentials: %v", err)
	}
	// Remove injected binaries before creating layer
	// so they don't pollute the image layer
	cleanupInjected(state.RootFS)

	return createLayerFromState(state, inst.Raw)
}

func cleanupInjected(root string) {
	// Remove binaries injected by RunIsolated/copyEssentials
	toRemove := []string{
		filepath.Join(root, "bin", "sh"),
		filepath.Join(root, "bin", "busybox"),
		filepath.Join(root, "lib"),
		filepath.Join(root, "lib64"),
		filepath.Join(root, "usr", "lib"),
		filepath.Join(root, "proc"),
	}
	for _, p := range toRemove {
		os.RemoveAll(p)
	}
}

func createLayerFromState(state *BuildState, instruction string) error {
	tempTar := filepath.Join(os.TempDir(), fmt.Sprintf("layer-%d.tar", time.Now().UnixNano()))

	// Compute current file hashes
	currentHashes, err := utils.ComputeFileHashes(state.RootFS)
	if err != nil {
		return err
	}

	// Use delta tar if we have previous state, otherwise use full tar
	if len(state.PreviousFileHashes) > 0 {
		if err := utils.CreateDeltaTar(state.RootFS, tempTar, state.PreviousFileHashes, currentHashes); err != nil {
			return err
		}
	} else {
		if err := utils.CreateTar(state.RootFS, tempTar); err != nil {
			return err
		}
	}

	digest, err := utils.HashFile(tempTar)
	if err != nil {
		return err
	}

	if _, err = storage.SaveLayer(tempTar, digest); err != nil {
		return err
	}

	recordProducedLayer(state, digest)
	// Store instruction text for this layer
	state.LayerCreatedBy[digest] = instruction
	// Update previous hashes for next layer
	state.PreviousFileHashes = currentHashes
	return nil
}
