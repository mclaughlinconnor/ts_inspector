package parser

import (
	"context"
	"log"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type parseCallback[V any] func(root *sitter.Node, content []byte, filename string, file V) (V, error)

func HandleFile(state State, uri string, languageId string, version int, logger *log.Logger) (State, error) {
	previousFile := state[filenameFromUri(uri)]

	var file File
	if languageId == "" {
		file = NewFile(uri, previousFile.Filetype, version)
	} else {
		file = NewFile(uri, languageId, version)
	}

	var err error

	filename := filenameFromUri(uri)
	if file.Filetype == "typescript" {
		file, err = HandleTypeScriptFile(filename, file, logger)
	} else if file.Filetype == "pug" {
		file, err = HandlePugFile(filename, file)
	}

	if err == nil {
		state[filename] = file
	}

	templateFilename := file.Template
	if templateFilename != "" {
		existingPugFile, found := state[templateFilename]
		var pugFile File

		if found {
			pugFile = NewFile(existingPugFile.URI, existingPugFile.Filetype, existingPugFile.Version)
		} else {
			filetype, err := FiletypeFromFilename(templateFilename)
			if err != nil {
				return state, err
			}

			pugFile = NewFile(uriFromFilename(templateFilename), filetype, 0)
		}

		pugFile, err = HandlePugFile(templateFilename, pugFile)
		if err == nil {
			state[templateFilename] = pugFile
		}
	}

	return state, err
}

func HandleTypeScriptFile(filename string, file File, logger *log.Logger) (File, error) {
	return parseFileContent(filename, TypeScript, file,
		parseCallback[File](func(root *sitter.Node, content []byte, filename string, file File) (File, error) {
			file, err := ExtractTypeScriptUsages(file, root, content)
			if err != nil {
				return file, err
			}

			file, err = ExtractTypeScriptDefinitions(file, root, content)
			if err != nil {
				return file, err
			}

			file, err = ExtractTemplateFilename(file, filename, root, content)
			if err != nil {
				return file, err
			}

			return file, nil
		}))
}

func HandlePugFile(filename string, file File) (File, error) {
	return parseFileContent(filename, Pug, file,
		parseCallback[File](func(root *sitter.Node, content []byte, filename string, file File) (File, error) {
			file, err := ExtractPugUsages(file, content)
			if err != nil {
				return file, err
			}
			return file, nil
		}))
}

func parseFileContent[V any](filename string, language string, file V, callback parseCallback[V]) (V, error) {
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

func filenameFromUri(uri string) string {
	return strings.TrimPrefix(uri, `file://`)
}

func uriFromFilename(filename string) string {
	return `file://` + filename
}
