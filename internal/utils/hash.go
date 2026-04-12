package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

func HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}