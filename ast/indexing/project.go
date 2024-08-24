package indexing

import (
	"log"
	"os"
	"path/filepath"
	"ts_inspector/utils"
)

func findProjectRoots(root string) []string {
	rootFiles, err := os.ReadDir(root)
	if err != nil {
		log.Fatal(err)
	}

	var tsConfig TsConfig = TsConfig{
		CompilerOptions: tsConfigCompilerOptions{
			BaseURL: []string{},
		},
		Exclude: []string{},
		Files:   []string{},
		Include: []string{},
	}

	readTsConfigs(root, rootFiles, &tsConfig)

	if len(tsConfig.CompilerOptions.BaseURL) != 0 {
		for _, baseUrl := range tsConfig.CompilerOptions.BaseURL {
			basePath := filepath.Join(root, baseUrl)
			rootFiles, err = os.ReadDir(basePath)
			if err != nil {
				log.Fatal(err)
			}
			readTsConfigs(basePath, rootFiles, &tsConfig)
		}
	}

	for i, f := range tsConfig.Files {
		tsConfig.Files[i] = filepath.Join(root, f)
	}

	files := []string{}

	for _, baseURL := range tsConfig.CompilerOptions.BaseURL {
		rootFiles, err := os.ReadDir(filepath.Join(root, baseURL))

		filenames := []string{}

		for _, rootFile := range rootFiles {
			filenames = append(filenames, filepath.Join(root, baseURL, rootFile.Name()))
		}

		if err == nil {
			files = append(files, utils.RemoveNonCode(filenames)...)
		}
	}

	return append(files, tsConfig.Files...)
}
