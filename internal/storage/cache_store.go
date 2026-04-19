package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var cacheFile = "cache_index.json"

func getCachePath() string {
	return filepath.Join(CacheDir(), cacheFile)
}

func LoadCache() (map[string]string, error) {
	cache := make(map[string]string)

	path := getCachePath()

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cache, nil
		}
		return nil, err
	}

	defer file.Close()

	err = json.NewDecoder(file).Decode(&cache)
	return cache, err
}

func SaveCache(cache map[string]string) error {
	err := os.MkdirAll(CacheDir(), 0755)
	if err != nil {
		return err
	}

	path := getCachePath()

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(cache)
}
