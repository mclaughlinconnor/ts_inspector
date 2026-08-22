package search

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"embed"
	"io"
	"os"
	"path/filepath"
	"ts_inspector/utils"
)

// Use github.com/hybridgroup/yzma/pkg/download to download the libs

//go:embed lib.tar.gz
var libTgzFs embed.FS

// investigate https://github.com/stephantul/pynife and https://huggingface.co/blobbybob/potion-mxbai-128d-v2

//go:embed granite-embedding-30m-english-Q4_K_S.gguf
var modelFs embed.FS

var modelName = "granite-embedding-30m-english-Q4_K_S.gguf"

func getCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".cache", "ts_inspector"), nil
}

func extractEmbeddedModel() (string, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(cacheDir, "models")
	path := filepath.Join(dir, modelName)

	_, err = os.Stat(path)
	if err == nil { // file found
		return path, nil
	}

	f, err := modelFs.Open(modelName)
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(f)

	err = makeFile(reader, path, 0755)
	if err != nil {
		return "", err
	}

	if err := f.Close(); err != nil {
		return "", err
	}

	return path, nil
}

func extractEmbeddedLibs() (string, error) {
	tempDir, err := os.MkdirTemp("", "ts_inspector-libs-*")
	if err != nil {
		return "", err
	}

	f, err := libTgzFs.Open("lib.tar.gz")
	if err != nil {
		return "", err
	}

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		path := filepath.Join(tempDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			err := makeDir(path)
			if err != nil {
				return "", err
			}
		case tar.TypeReg:
			err := makeFile(tarReader, path, header.Mode)
			if err != nil {
				return "", err
			}
		case tar.TypeSymlink:
			err := makeSymlink(header.Linkname, path)
			if err != nil {
				return "", err
			}
		}
	}

	if err := f.Close(); err != nil {
		return "", err
	}

	if err := gzReader.Close(); err != nil {
		return "", err
	}

	return filepath.Join(tempDir, "lib"), nil
}

func makeDir(path string) error {
	err := os.MkdirAll(path, 0755)
	return err
}

func makeFile(reader io.Reader, path string, mode int64) error {
	err := os.MkdirAll(utils.PathDir(path), 0755)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.FileMode(mode))
	if err != nil {
		return err
	}

	_, err = io.Copy(f, reader)
	if err != nil {
		return err
	}

	err = f.Close()

	return err
}

func makeSymlink(linkName string, path string) error {
	err := os.MkdirAll(utils.PathDir(path), 0755)
	if err != nil {
		return err
	}

	err = os.Symlink(linkName, path)
	return err
}
