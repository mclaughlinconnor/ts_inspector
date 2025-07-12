package utils

import (
	"dataset_gen/treesitter_parsers/angular_content"
	"dataset_gen/treesitter_parsers/javascript"
	"dataset_gen/treesitter_parsers/pug"
	"dataset_gen/treesitter_parsers/typescript"

	sitter "github.com/smacker/go-tree-sitter"
)

const (
	AngularContent = "angular_content"
	Pug            = "pug"
	TypeScript     = "typescript"
	JavaScript     = "javascript"
)

var languageConsts = []string{AngularContent, JavaScript, Pug, TypeScript}

var languages = map[string]*sitter.Language{
	AngularContent: angular_content.GetLanguage(),
	Pug:            pug.GetLanguage(),
	TypeScript:     typescript.GetLanguage(),
	JavaScript:     javascript.GetLanguage(),
}

func GetLanguage(language string) *sitter.Language {
	return languages[language]
}
