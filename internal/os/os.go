package os

import (
	"os"
	"path/filepath"
)

var customConfigDir string

func SetCustomConfigDir(dir string) {
	customConfigDir = dir
}

func CachePath(subDir string) string {
	return filepath.Join(ConfigPath(), "cache", subDir)
}

func ConfigPath() string {
	var configDir string
	if len(customConfigDir) > 0 {
		configDir = customConfigDir
	} else {
		dir, _ := os.UserConfigDir()
		configDir = filepath.Join(dir, "gungus")
	}

	return configDir
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func CreateFile(path string) error {
	dir, _ := filepath.Split(path)

	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	return file.Close()
}
