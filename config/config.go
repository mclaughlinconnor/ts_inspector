package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"ts_inspector/utils"
)

var config Config

type ConfigIndexing struct {
	ExperiementalParallelInitialIndexing bool `json:"experiementalParallelInitialIndexing"`
}

type ConfigLsp struct {
	Enable bool `json:"enable"`
}

type ConfigSemanticSearch struct {
	Enable                       bool `json:"enable"`
	EnableFaiss                  bool `json:"enableFaiss"`
	EnableFzf                    bool `json:"enableFzf"`
	EnableSqlite                 bool `json:"enableSqlite"`
	IncludeFileInterestingPoints bool `json:"includeFileInterestingPoints"`
}

type ConfigTcb struct {
	ExperimentalTagBasedAttributeRendering bool `json:"experimentalTagBasedAttributeRendering"`
}

type ConfigTsGo struct {
	Enable                                bool   `json:"enable"`
	ExperimentalThingCaching              bool   `json:"experimentalThingCaching"`
	ExperimentalConcurrentRequestHandling bool   `json:"experimentalConcurrentRequestHandling"`
	BinaryPath                            string `json:"binaryPath"`
}

type Config struct {
	Concurrency    bool                 `json:"concurrency"`
	Debug          bool                 `json:"debug"`
	DelayStart     bool                 `json:"delayStart"`
	Indexing       ConfigIndexing       `json:"indexing"`
	LSP            ConfigLsp            `json:"lsp"`
	LogsPath       string               `json:"logsPath"`
	MinimalStartup bool                 `json:"minimalStartup"`
	SemanticSearch ConfigSemanticSearch `json:"semanticSearch"`
	Tcb            ConfigTcb            `json:"tcb"`
	TsGo           ConfigTsGo           `json:"tsGo"`
}

func getConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

func getDefaultConfig() Config {
	config := Config{}

	config.Concurrency = false
	config.Debug = false
	config.DelayStart = false
	config.Indexing.ExperiementalParallelInitialIndexing = false
	config.LSP.Enable = true
	config.LogsPath = "/home/connor/Development/ts_inspector/logs/"
	config.SemanticSearch.Enable = true
	config.SemanticSearch.EnableFaiss = false
	config.SemanticSearch.EnableFzf = true
	config.SemanticSearch.EnableSqlite = true
	config.SemanticSearch.IncludeFileInterestingPoints = false
	config.Tcb.ExperimentalTagBasedAttributeRendering = false
	config.TsGo.Enable = true
	config.TsGo.ExperimentalThingCaching = false
	config.TsGo.ExperimentalConcurrentRequestHandling = false
	config.TsGo.BinaryPath = "/home/connor/.local/share/nvim/mason/packages/tsgo/node_modules/@typescript/native-preview/lib/tsgo.js"

	return config
}

func loadConfigFileContent(configPath string) ([]byte, error) {
	var content []byte
	var err error

	if utils.FileExists(configPath) {
		content, err = os.ReadFile(configPath)
		if err != nil {
			return content, err
		}

		return content, nil
	}

	content, err = json.MarshalIndent(getDefaultConfig(), "", "  ")
	if err != nil {
		return content, err
	}

	err = os.WriteFile(configPath, content, os.ModePerm)
	if err != nil {
		return content, err
	}

	return content, nil
}

func postProcessConfig() {
	if config.MinimalStartup {
		config.DelayStart = false
		config.SemanticSearch.Enable = false
		config.TsGo.Enable = false
	}
}

func GetConfig() *Config {
	return &config
}

func InitConfig() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	configContent, err := loadConfigFileContent(configPath)
	if err != nil {
		return err
	}

	loadedConfig := getDefaultConfig()

	err = json.Unmarshal(configContent, &loadedConfig)
	if err != nil {
		return err
	}

	postProcessConfig()

	config = loadedConfig

	return nil
}
