package utils

import (
	"archive/tar"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func CreateTar(sourceDir, tarPath string) error {
	tarFile, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	tw := tar.NewWriter(tarFile)
	defer tw.Close()

	var paths []string
	err = filepath.Walk(sourceDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}

		// Skip the tar file itself
		if file == tarPath {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(sourceDir, file)
		if err != nil {
			return nil
		}

		// Skip virtual/special directories
		skipDirs := []string{"proc", "sys", "dev", "run"}
		for _, skip := range skipDirs {
			if relPath == skip || strings.HasPrefix(relPath, skip+"/") {
				if fi.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		paths = append(paths, file)
		return nil
	})
	if err != nil {
		return err
	}

	sort.Strings(paths)

	for _, file := range paths {
		fi, err := os.Lstat(file)
		if err != nil {
			continue
		}

		header, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			continue
		}

		relPath, err := filepath.Rel(sourceDir, file)
		if err != nil {
			continue
		}
		header.Name = relPath
		header.ModTime = time.Time{}
		header.AccessTime = time.Time{}
		header.ChangeTime = time.Time{}

		if !fi.IsDir() {
			realFi, err := os.Stat(file)
			if err != nil {
				continue
			}
			header.Size = realFi.Size()
		}

		if err := tw.WriteHeader(header); err != nil {
			continue
		}

		if fi.IsDir() {
			continue
		}

		f, err := os.Open(file)
		if err != nil {
			continue
		}
		_, err = io.Copy(tw, f)
		f.Close()
		if err != nil {
			continue
		}
	}

	return nil
}

// ComputeFileHashes returns a map of relative paths to MD5 hashes of all files in a directory
func ComputeFileHashes(sourceDir string) (map[string]string, error) {
	hashes := make(map[string]string)
	
	err := filepath.Walk(sourceDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(sourceDir, file)
		if err != nil {
			return nil
		}

		// Skip virtual/special directories
		skipDirs := []string{"proc", "sys", "dev", "run"}
		for _, skip := range skipDirs {
			if relPath == skip || strings.HasPrefix(relPath, skip+"/") {
				if fi.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if fi.IsDir() {
			return nil
		}

		// Compute MD5 hash of file
		f, err := os.Open(file)
		if err != nil {
			return nil
		}
		defer f.Close()

		hasher := md5.New()
		if _, err := io.Copy(hasher, f); err != nil {
			return nil
		}

		hashes[relPath] = fmt.Sprintf("%x", hasher.Sum(nil))
		return nil
	})

	return hashes, err
}

// CreateDeltaTar creates a tar archive with only files that changed between two snapshots
// previousHashes: map from relative path to hash before instruction
// currentHashes: map from relative path to hash after instruction
// sourceDir: the root directory to tar from
// tarPath: destination tar file path
func CreateDeltaTar(sourceDir, tarPath string, previousHashes, currentHashes map[string]string) error {
	tarFile, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	tw := tar.NewWriter(tarFile)
	defer tw.Close()

	// Find changed files (new or modified)
	var changedPaths []string
	for relPath, newHash := range currentHashes {
		oldHash, existed := previousHashes[relPath]
		// Include if new or if modified (hash different)
		if !existed || oldHash != newHash {
			changedPaths = append(changedPaths, filepath.Join(sourceDir, relPath))
		}
	}

	// Sort for deterministic ordering
	sort.Strings(changedPaths)

	// Write changed files to tar
	for _, file := range changedPaths {
		fi, err := os.Lstat(file)
		if err != nil {
			continue
		}

		header, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			continue
		}

		relPath, err := filepath.Rel(sourceDir, file)
		if err != nil {
			continue
		}
		header.Name = relPath
		header.ModTime = time.Time{}
		header.AccessTime = time.Time{}
		header.ChangeTime = time.Time{}

		if !fi.IsDir() {
			realFi, err := os.Stat(file)
			if err != nil {
				continue
			}
			header.Size = realFi.Size()
		}

		if err := tw.WriteHeader(header); err != nil {
			continue
		}

		if fi.IsDir() {
			continue
		}

		f, err := os.Open(file)
		if err != nil {
			continue
		}
		_, err = io.Copy(tw, f)
		f.Close()
		if err != nil {
			continue
		}
	}

	return nil
}
