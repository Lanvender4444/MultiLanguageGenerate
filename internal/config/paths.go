package config

import (
	"os"
	"path/filepath"
)

func ConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "MultiLanguageGenerate")
	return path, os.MkdirAll(path, 0755)
}

func ConfigFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}
