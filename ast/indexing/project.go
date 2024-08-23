package indexing

import (
	"log"
	"os"
	"path/filepath"
)

func findProjectRoots(root string) []string {
	rootFiles, err := os.ReadDir(root)
	if err != nil {
		log.Fatal(err)
	}

	tsConfig := readTsConfigs(root, rootFiles)

	for i, f := range tsConfig.Files {
		tsConfig.Files[i] = filepath.Join(root, f)
	}

	return tsConfig.Files // TODO: handle include & exclude
}
