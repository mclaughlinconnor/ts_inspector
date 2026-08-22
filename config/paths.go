package config

import (
	"os"
	"path/filepath"
)

func getDir(homeDirDir string, subDirs ...string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	pathParts := make([]string, 0, 3+len(subDirs))
	pathParts = append(pathParts, home, homeDirDir, "ts_inspector")
	pathParts = append(pathParts, subDirs...)

	path := filepath.Join(pathParts...)

	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return "", err
	}

	return path, nil
}

func GetConfigDir(subDirs ...string) (string, error) {
	return getDir(".config", subDirs...)
}

func GetCacheDir(subDirs ...string) (string, error) {
	return getDir(".cache", subDirs...)
}
