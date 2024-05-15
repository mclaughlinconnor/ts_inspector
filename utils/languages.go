package utils

import (
	"ts_inspector/treesitter_parsers/angular_content"
	"ts_inspector/treesitter_parsers/javascript"
	"ts_inspector/treesitter_parsers/pug"
	"ts_inspector/treesitter_parsers/typescript"

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
