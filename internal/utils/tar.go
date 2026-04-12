package utils

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
	"strings"
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
