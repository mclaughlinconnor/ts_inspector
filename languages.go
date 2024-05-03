package main

import (
	"github.com/mclaughlinconnor/ts_inspector/parsers/angular_content"
	"github.com/mclaughlinconnor/ts_inspector/parsers/javascript"
	"github.com/mclaughlinconnor/ts_inspector/parsers/pug"
	"github.com/mclaughlinconnor/ts_inspector/parsers/typescript"
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
