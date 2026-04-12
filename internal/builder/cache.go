package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"docksmith/internal/model"
)

func generateCacheKey(inst model.Instruction, state *BuildState) string {
	var data strings.Builder

	// Previous layer digest (or empty if none yet)
	if len(state.Layers) > 0 {
		data.WriteString(state.Layers[len(state.Layers)-1])
	}

	// Full instruction text
	data.WriteString(inst.Raw)

	// Current WORKDIR
	data.WriteString(state.WorkingDir)

	// ENV state: sorted by key
	keys := make([]string, 0, len(state.Env))
	for k := range state.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		data.WriteString(k + "=" + state.Env[k])
	}

	// COPY only: hash each source file in sorted order
	if inst.Type == "COPY" && len(inst.Args) >= 1 {
		src := inst.Args[0]
		filePaths := []string{}
		filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				filePaths = append(filePaths, path)
			}
			return nil
		})
		sort.Strings(filePaths)
		for _, path := range filePaths {
			h, err := hashFile(path)
			if err == nil {
				data.WriteString(path + ":" + h)
			}
		}
	}

	hash := sha256.Sum256([]byte(data.String()))
	return hex.EncodeToString(hash[:])
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
