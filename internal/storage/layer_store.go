package storage

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func SaveLayer(tempTarPath, digest string) (string, error) {

	safeDigest := strings.ReplaceAll(digest, ":", "_")

	filename := safeDigest + ".tar"
	finalPath := filepath.Join(LayersDir(), filename)

	err := os.MkdirAll(LayersDir(), 0755)
	if err != nil {
		return "", err
	}

	err = copyFile(tempTarPath, finalPath)
	if err != nil {
		return "", err
	}

	err = os.Remove(tempTarPath)
	if err != nil {
		return "", err
	}
	return finalPath, nil
}

func LayerSize(filename string) int64 {
	path := filepath.Join(LayersDir(), filename)
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func RemoveLayer(filename string) error {
	path := filepath.Join(LayersDir(), filename)
	return os.Remove(path)
}
