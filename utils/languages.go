package utils

import (
	"ts_inspector/treesitter_parsers/angular_content"
	"ts_inspector/treesitter_parsers/angular_expr"
	"ts_inspector/treesitter_parsers/javascript"
	"ts_inspector/treesitter_parsers/markdown"
	"ts_inspector/treesitter_parsers/pug"
	"ts_inspector/treesitter_parsers/typescript"
	"ts_inspector/treesitter_parsers/yaml"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

const (
	AngularContent = "angular_content"
	AngularExpr    = "angular_expr"
	JavaScript     = "javascript"
	Markdown       = "markdown"
	Pug            = "pug"
	TypeScript     = "typescript"
	Yaml           = "yaml"
)

var languages = map[string]*sitter.Language{
	AngularContent: angular_content.GetLanguage(),
	AngularExpr:    angular_expr.GetLanguage(),
	JavaScript:     javascript.GetLanguage(),
	Markdown:       markdown.GetLanguage(),
	Pug:            pug.GetLanguage(),
	TypeScript:     typescript.GetLanguage(),
	Yaml:           yaml.GetLanguage(),
}

func GetLanguage(language string) *sitter.Language {
	return languages[language]
}
