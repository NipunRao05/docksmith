package utils

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
)

func ExtractTar(tarPath, targetDir string) error {

	file, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer file.Close()

	tr := tar.NewReader(file)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		target := filepath.Join(targetDir, header.Name)

		if header.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(target), 0755)

		out, err := os.Create(target)
		if err != nil {
			return err
		}

		_, err = io.Copy(out, tr)
		out.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
