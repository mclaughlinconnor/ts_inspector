package parser

import (
	"log"

	sitter "github.com/smacker/go-tree-sitter"
)

type parseCallback[V any] func(root *sitter.Node, content []byte, v V) (V, error)

var lastSeenDocumentVersion = make(map[string]int, 0)

func handleFile(state *State, uri string, languageId string, version int, content string, _ *log.Logger) (File, error) {
	v := version
	if v == 0 {
		lastSeenVersion, found := lastSeenDocumentVersion[uri]
		if found {
			v = lastSeenVersion
		}
	} else {
		lastSeenDocumentVersion[uri] = v
	}

	file, err := NewFile(uri, languageId, v)
	if err != nil {
		return file, err
	}

	file.SetContent(content)

	if languageId == "typescript" {
		file, err = HandleTypeScriptFile(state, file)
	}

	return file, err
}

func HandleFile(state State, uri string, languageId string, version int, content string, logger *log.Logger) (State, error) {
	if languageId == "" {
		var err error
		languageId, err = FiletypeFromFilename(FilenameFromUri(uri))
		if err != nil {
			return state, err
		}
	}

	file, err := handleFile(&state, uri, languageId, version, content, logger)
	if err != nil {
		return state, err
	}

	state.Files[file.Filename()] = &file

	for _, class := range file.Classes {
		state.Classes[class.Id()] = class
	}

	state, err = handleDependencies(file, state, logger)
	if err != nil {
		return state, err
	}

	state.ResolveClassRelationships()

	return state, nil
}

func handleDependencies(file File, state State, logger *log.Logger) (State, error) {
	dependencies := file.GetDependencies()
	for _, depFile := range dependencies {
		state, err := handleDependency(state, depFile, logger)

		if err != nil {
			return state, err
		}
	}

	return state, nil
}

func handleDependency(state State, filename string, logger *log.Logger) (State, error) {
	filetype, err := FiletypeFromFilename(filename)
	if err != nil {
		return state, err
	}
	df, err := handleFile(&state, UriFromFilename(filename), filetype, 0, state.Files[filename].Content, logger)
	if err != nil {
		return state, err
	}
	state.Files[df.Filename()] = &df

	for _, class := range df.Classes {
		state.Classes[class.Id()] = class
	}

	return state, nil
}
