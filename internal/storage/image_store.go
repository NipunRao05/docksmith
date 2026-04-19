package storage

import (
	"encoding/json"
	"os"
	"path/filepath"

	"docksmith/internal/model"
)

func SaveImage(img model.Image) error {

	err := os.MkdirAll(ImagesDir(), 0755)
	if err != nil {
		return err
	}

	filename := img.Name + "_" + img.Tag + ".json"
	path := filepath.Join(ImagesDir(), filename)

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(img)
}

func ListImages() ([]string, error) {

	err := os.MkdirAll(ImagesDir(), 0755)
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(ImagesDir())
	if err != nil {
		return nil, err
	}

	var images []string
	for _, file := range files {
		if !file.IsDir() {
			images = append(images, file.Name())
		}
	}

	return images, nil
}

func RemoveImage(name string) error {

	path := filepath.Join(ImagesDir(), name)

	err := os.Remove(path)
	if err != nil {
		return err
	}

	return nil
}

func LoadImage(name string) (*model.Image, error) {

	path := filepath.Join(ImagesDir(), name)

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var img model.Image
	err = json.NewDecoder(file).Decode(&img)
	if err != nil {
		return nil, err
	}

	return &img, nil
}
