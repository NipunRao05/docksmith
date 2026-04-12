package storage

import (
	"os"
	"path/filepath"
)

var baseDir string

func init() {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		// fallback for sudo / missing HOME
		home = "/root"
	}

	baseDir = filepath.Join(home, ".docksmith")

	// Ensure all required directories exist
	os.MkdirAll(filepath.Join(baseDir, "images"), 0755)
	os.MkdirAll(filepath.Join(baseDir, "layers"), 0755)
	os.MkdirAll(filepath.Join(baseDir, "cache"), 0755)
}

func BaseDir() string {
	return baseDir
}

func ImagesDir() string {
	return filepath.Join(baseDir, "images")
}

func LayersDir() string {
	return filepath.Join(baseDir, "layers")
}

func CacheDir() string {
	return filepath.Join(baseDir, "cache")
}

