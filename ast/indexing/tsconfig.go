package indexing

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"ts_inspector/utils"

	"github.com/tailscale/hujson"
)

type TsConfig struct {
	Exclude []string `json:"exclude"`
	Files   []string `json:"files"`
	Include []string `json:"include"`
}

func parseTsConfig(filename string) TsConfig {
	content, err := utils.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	var config TsConfig
	content, err = hujson.Standardize(bytes.Trim(content, "\x00"))
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(content, &config)

	return config
}

func readTsConfigs(root string, rootFiles []fs.DirEntry) TsConfig {
	var tsConfig TsConfig = TsConfig{
		Exclude: []string{},
		Files:   []string{},
		Include: []string{},
	}

	for _, rootFile := range rootFiles {
		rootFileName := rootFile.Name()
		if strings.HasPrefix(rootFileName, "tsconfig.") && strings.HasSuffix(rootFileName, ".json") {
			config := parseTsConfig(filepath.Join(root, rootFileName))
			tsConfig.Exclude = append(tsConfig.Exclude, config.Exclude...)
			tsConfig.Files = append(tsConfig.Files, config.Files...)
			tsConfig.Include = append(tsConfig.Include, config.Include...)
		}
	}

	tsConfig.Exclude = utils.RemoveNonCode(tsConfig.Exclude)
	tsConfig.Files = utils.RemoveNonCode(tsConfig.Files)
	tsConfig.Include = utils.RemoveNonCode(tsConfig.Include)

	return tsConfig
}
