package parser

import (
	"context"
	"log"

	sitter "github.com/smacker/go-tree-sitter"
)

type parseCallback[V any] func(root *sitter.Node, content []byte, filename string, file V) (V, error)

func HandleFile(state State, uri string, languageId string, version int, content string, logger *log.Logger) (State, error) {
	previousFile := state[FilenameFromUri(uri)]

	var file File
	if languageId == "" {
		file = NewFile(uri, previousFile.Filetype, version)
	} else {
		file = NewFile(uri, languageId, version)
	}

	file.Content = content

	var err error

	filename := FilenameFromUri(uri)
	if file.Filetype == "typescript" {
		file, err = HandleTypeScriptFile(file, logger)
	} else if file.Filetype == "pug" {
		file, err = HandlePugFile(file)
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

			pugFile = NewFile(UriFromFilename(templateFilename), filetype, 0)
		}

		pugFile, err = HandlePugFile(pugFile)
		if err == nil {
			state[templateFilename] = pugFile
		}
	}

	return state, err
}

func HandleTypeScriptFile(file File, logger *log.Logger) (File, error) {
	fromDisk := file.Content == ""
	var source string
	if fromDisk {
		source = file.Filename()
	} else {
		source = file.Content
	}

	return parseFile(fromDisk, source, TypeScript, file,
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

func HandlePugFile(file File) (File, error) {
	fromDisk := file.Content == ""
	var source string
	if fromDisk {
		source = file.Filename()
	} else {
		source = file.Content
	}

	return parseFile(fromDisk, source, Pug, file,
		parseCallback[File](func(root *sitter.Node, content []byte, filename string, file File) (File, error) {
			file, err := ExtractPugUsages(file, content)
			if err != nil {
				return file, err
			}
			return file, nil
		}))
}

func parseFile[V any](fromDisk bool, source string, language string, file V, callback parseCallback[V]) (V, error) {
	var content []byte
	if fromDisk {
		content = ReadFile(source)
	} else {
		content = []byte(source)
	}

	parser := sitter.NewParser()
	parser.SetLanguage(GetLanguage(language))

	tree, err := parser.ParseCtx(context.TODO(), nil, content)
	if err != nil {
		log.Fatal(err)
	}

	root := tree.RootNode()

	return callback(root, content, source, file)
}
