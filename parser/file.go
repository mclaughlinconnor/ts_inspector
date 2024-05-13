package parser

import (
	"context"
	"log"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type parseCallback[V any] func(root *sitter.Node, content []byte, filename string, file V) (result V, ok bool)

func HandleFile(state State, filename string, languageId string, _ *log.Logger) (State, bool) {
	var file File = NewFile()
	var ok bool

	filename = stripFileName(filename)

	if languageId == "typescript" {
		file, ok = HandleTypeScriptFile(filename, file)
	} else if languageId == "pug" {
		file, ok = HandlePugFile(filename, file)
	}

	if ok {
		state[filename] = file
	}

	templateFilename := file.Template
	if templateFilename != "" {
		pugFile := NewFile()
		pugFile, ok = HandlePugFile(templateFilename, pugFile)
		if ok {
			state[templateFilename] = pugFile
		}
	}

	return state, ok
}

func HandleTypeScriptFile(filename string, file File) (File, bool) {
	return parseFileContent(filename, TypeScript, file,
		parseCallback[File](func(root *sitter.Node, content []byte, filename string, file File) (result File, ok bool) {
			file, err := ExtractTypeScriptUsages(file, root, content)
			if err != nil {
				log.Print(err)
			}

			file, err = ExtractTypeScriptDefinitions(file, root, content)
			if err != nil {
				log.Print(err)
			}

			file, err = ExtractTemplateFilename(file, filename, root, content)
			if err != nil {
				log.Fatal(err)
			}

			return file, true
		}))
}

func HandlePugFile(filename string, file File) (File, bool) {
	return parseFileContent(filename, Pug, file,
		parseCallback[File](func(root *sitter.Node, content []byte, filename string, file File) (result File, ok bool) {
			file, err := ExtractPugUsages(file, content)
			if err != nil {
				return file, false
			}
			return file, true
		}))
}

func parseFileContent[V any](filename string, language string, file V, callback parseCallback[V]) (result V, ok bool) {
	content := ReadFile(filename)

	parser := sitter.NewParser()
	parser.SetLanguage(GetLanguage(language))

	tree, err := parser.ParseCtx(context.TODO(), nil, content)
	if err != nil {
		log.Fatal(err)
	}

	root := tree.RootNode()

	return callback(root, content, filename, file)
}

func stripFileName(filename string) string {
	return strings.TrimPrefix(filename, `file://`)
}
