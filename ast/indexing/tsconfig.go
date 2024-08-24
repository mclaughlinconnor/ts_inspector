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

type tsConfigCompilerOptionsFile struct {
	BaseURL string `json:"baseUrl"`
}

type tsConfigCompilerOptions struct {
	BaseURL []string `json:"baseUrl"`
}

// TODO: maybe there's a better way of embedding this
type TsConfigFile struct {
	Exclude         []string                    `json:"exclude"`
	Files           []string                    `json:"files"`
	Include         []string                    `json:"include"`
	CompilerOptions tsConfigCompilerOptionsFile `json:"compilerOptions"`
}

type TsConfig struct {
	Exclude         []string                `json:"exclude"`
	Files           []string                `json:"files"`
	Include         []string                `json:"include"`
	CompilerOptions tsConfigCompilerOptions `json:"compilerOptions"`
}

func parseTsConfig(filename string) TsConfigFile {
	content, err := utils.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	var config TsConfigFile
	content, err = hujson.Standardize(bytes.Trim(content, "\x00"))
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(content, &config)

	return config
}

func readTsConfigs(root string, rootFiles []fs.DirEntry, tsConfig *TsConfig) {
	for _, rootFile := range rootFiles {
		rootFileName := rootFile.Name()
		if strings.HasPrefix(rootFileName, "tsconfig.") && strings.HasSuffix(rootFileName, ".json") {
			config := parseTsConfig(filepath.Join(root, rootFileName))
			tsConfig.Exclude = append(tsConfig.Exclude, config.Exclude...)
			tsConfig.Files = append(tsConfig.Files, config.Files...)
			tsConfig.Include = append(tsConfig.Include, config.Include...)

			if config.CompilerOptions.BaseURL != "" {
				tsConfig.CompilerOptions.BaseURL = append(tsConfig.CompilerOptions.BaseURL, config.CompilerOptions.BaseURL)
			}
		}
	}

	tsConfig.Exclude = utils.RemoveNonCode(tsConfig.Exclude)
	tsConfig.Files = utils.RemoveNonCode(tsConfig.Files)
	tsConfig.Include = utils.RemoveNonCode(tsConfig.Include)
}
