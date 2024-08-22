package parser

import (
	"log"

	sitter "github.com/smacker/go-tree-sitter"
)

type parseCallback[V any] func(root *sitter.Node, content []byte, v V) (V, error)

func handleFile(uri string, languageId string, version int, content string, _ *log.Logger) (File, error) {
	file, err := NewFile(uri, languageId, version)
	if err != nil {
		return file, err
	}

	file = file.SetContent(content)

	if languageId == "typescript" {
		file, err = HandleTypeScriptFile(file)
	} else if languageId == "pug" {
		file, err = HandlePugFile(file)
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

	file, err := handleFile(uri, languageId, version, content, logger)
	if err != nil {
		return state, err
	}

	state.Files[file.Filename()] = file

	state, err = handleDependencies(file, state, logger)
	if err != nil {
		return state, err
	}

	state = reconcile(state)

	return state, nil
}

func handleDependencies(file File, state State, logger *log.Logger) (State, error) {
	filename := file.Filename()

	for fn, f := range state.Files {
		var err error
		if f.Template == filename || f.Controller == filename {
			state, err = handleDependency(state, fn, logger)
		}
		if fn == filename {
			if f.Template != "" {
				state, err = handleDependency(state, f.Template, logger)
			}
			if f.Controller != "" {
				state, err = handleDependency(state, f.Controller, logger)
			}
		}

		if err != nil {
			return state, err
		}
	}

	for fn, f := range state.Files {
		if f.Template != "" {
			t := state.Files[f.Template]
			t.Controller = fn
			state.Files[f.Template] = t
		}
	}

	return state, nil
}

func handleDependency(state State, filename string, logger *log.Logger) (State, error) {
	filetype, err := FiletypeFromFilename(filename)
	if err != nil {
		return state, err
	}
	df, err := handleFile(UriFromFilename(filename), filetype, 0, state.Files[filename].Content, logger)
	if err != nil {
		return state, err
	}
	state.Files[df.Filename()] = df
	return state, nil
}

func reconcile(state State) State {
	for _, file := range state.Files {
		// Skip if is a controller
		if file.Controller != "" {
			continue
		}

		template := state.Files[file.Template]
		for name, usage := range template.Usages {
			for _, use := range usage.Usages {
				file = file.AppendDefinitionUsage(name, use)
			}
		}

		state.Files[file.Filename()] = file
	}

	return state
}
