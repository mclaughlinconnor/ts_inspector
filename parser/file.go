package parser

import (
	"log"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type parseCallback[V any] func(root *sitter.Node, content []byte, v V) (V, error)

func (s *State) Postprocess() {
	for _, file := range s.Files {
    file.Postprocess(s)
	}
}

func IndexFileFromLsp(state *State, uri string, languageId string, version int, content string, logger *log.Logger) error {
	var err error

	language := languageId

	if language == "" {
		filetype, err := FiletypeFromFilename(FilenameFromUri(uri))
		if err != nil {
			return err
		}

		language = filetype
	}

	if language == "typescript" {
		err = IndexTypeScriptFileFromLsp(state, uri, languageId, version, content, logger)
	} else if language == "pug" {
		err = IndexPugFileFromLsp(state, uri, content, version)
	}

	return err
}

func IndexFileFromIndexer(state *State, filename string) error {
	var err error

	filetype, err := FiletypeFromFilename(filename)
	if err != nil {
		return err
	}

	if filetype == "typescript" {
		err = IndexTypeScriptFileFromIndexer(state, filename)
	} else if filetype == "pug" {
		err = IndexPugFromIndexer(state, filename)
	}

  if err != nil {
    return err
  }

  file, found := state.Files[filename]
  if found {
    file.Postprocess(state)
  }

  return nil
}

func createFileIfNotExists(state *State, filename string, content string, version int) (*File, error) {
	file, found := state.Files[filename]
	if !found {
		uri := UriFromFilename(filename)

		filetype, err := FiletypeFromFilename(filename)
		if err != nil {
			return nil, err
		}

		file, err = NewFile(uri, filetype, versionFallback(0, uri))
		if err != nil {
			return nil, err
		}

		if content != "" {
			file.SetContent(content, version)
		} else {
			_, err = utils.ParseFile(true, file.Filename(), filetype, nil, func(root *sitter.Node, content []byte, _ any) (any, error) {
				file.SetContent(CStr2GoStr(content), version)

				return nil, nil
			})
		}

		state.Files[file.Filename()] = file
	} else {
		if content != "" || version != 0 {
			file.SetContent(content, version)
		}
	}

	return file, nil
}

func handleDependencies(file *File, state *State, logger *log.Logger) error {
	dependencies := file.GetDependencies(state)

	for _, depFile := range dependencies {
		err := IndexFileFromIndexer(state, depFile)

		if err != nil {
			return err
		}
	}

	return nil
}
