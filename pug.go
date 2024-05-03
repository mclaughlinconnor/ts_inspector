package main

import (
	"errors"
	"path"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractTemplateFilename(controllerFilePath string, root *sitter.Node, content []byte) (filename string, err error) {
	qc, q := GetQuery(QueryComponentDecorator, TypeScript)

	qc.Exec(q, root)

	for {
		m, ok := qc.NextMatch()
		if !ok {
			return "", errors.New("Could not match component decorator")
		}

		m = qc.FilterPredicates(m, content)
		if len(m.Captures) == 0 {
      continue
		}

		relativeTemplatePath := m.Captures[2].Node.Content(content)
		controllerDirectory := filepath.Dir(controllerFilePath)

		templateFilePath, err := filepath.Abs(path.Join(controllerDirectory, relativeTemplatePath))
		if err != nil {
			return "", err
		}

		if FileExists(templateFilePath) {
			return templateFilePath, nil
		}

		return "", errors.New("Expected template file does not exist")
	}
}
