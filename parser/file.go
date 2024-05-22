package parser

import (
	"log"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type parseCallback[V any] func(root *sitter.Node, content []byte, v V) (V, error)

func HandleFile(state State, uri string, languageId string, version int, content string, logger *log.Logger) (State, error) {
	previousFile, found := state[FilenameFromUri(uri)]

	var file File
	if languageId == "" {
		file = NewFile(uri, previousFile.Filetype, version, previousFile.Controller, previousFile.Template)
	} else {
		file = NewFile(uri, languageId, version, "", "")
	}

	if !found {
		state[file.Filename()] = file
	}

	if content != "" {
		file.Content = content
	}

	var err error

	if file.Filetype == "typescript" {
		state, err = HandleTypeScriptFile(file, state)
	} else if file.Filetype == "pug" {
		state, err = HandlePugFile(file, state)
	}

	file = state[file.Filename()]

	templateFilename := file.Template
	if templateFilename != "" {
		existingPugFile, found := state[templateFilename]
		var pugFile File

		if found {
			pugFile = NewFile(existingPugFile.URI, existingPugFile.Filetype, existingPugFile.Version, "", file.Filename())
		} else {
			filetype, err := FiletypeFromFilename(templateFilename)
			if err != nil {
				return state, err
			}

			// Do it here as well as in `ExtractTemplateFilename` because the pug file might not exist yet
			pugFile = NewFile(UriFromFilename(templateFilename), filetype, 0, "", file.Filename())
		}

		state[pugFile.Filename()] = pugFile
		state, err = HandlePugFile(pugFile, state)
	}

	controllerFilename := file.Controller
	if controllerFilename != "" {
		existingTsFile, found := state[controllerFilename]
		var controllerFile File

		if found {
			controllerFile = NewFile(existingTsFile.URI, existingTsFile.Filetype, existingTsFile.Version, "", file.Filename())
		} else {
			filetype, err := FiletypeFromFilename(templateFilename)
			if err != nil {
				return state, err
			}

			// Do it here as well as in `ExtractTemplateFilename` because the pug file might not exist yet
			controllerFile = NewFile(UriFromFilename(templateFilename), filetype, 0, "", file.Filename())
		}

		state[controllerFile.Filename()] = controllerFile
		state, err = HandleTypeScriptFile(controllerFile, state)
	}

	return state, err
}

func HandleTypeScriptFile(file File, state State) (State, error) {
	fromDisk := file.Content == ""
	var source string
	if fromDisk {
		source = file.Filename()
	} else {
		source = file.Content
	}

	return utils.ParseFile(fromDisk, source, utils.TypeScript, state,
		func(root *sitter.Node, content []byte, state State) (State, error) {
			file.Content = CStr2GoStr(content)
			state[file.Filename()] = file

			state, err := ExtractTypeScriptDefinitions(file, state, root, content)
			if err != nil {
				return state, err
			}

			state, err = ExtractTypeScriptUsages(file, state, root, content)
			if err != nil {
				return state, err
			}

			state, err = ExtractTemplateFilename(file, state, file.Filename(), root, content)
			if err != nil {
				return state, err
			}

			return state, nil
		})
}

func HandlePugFile(file File, state State) (State, error) {
	fromDisk := file.Content == ""
	var source string
	if fromDisk {
		source = file.Filename()
	} else {
		source = file.Content
	}

	return utils.ParseFile(fromDisk, source, utils.Pug, state,
		func(root *sitter.Node, content []byte, state State) (State, error) {
			file.Content = CStr2GoStr(content)
			state[file.Filename()] = file

			state, err := ExtractPugUsages(file, state, content)
			if err != nil {
				return state, err
			}

			return state, nil
		})
}
