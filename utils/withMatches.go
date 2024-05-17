package utils

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"
)

type HandleMatch[T any] func(captures []sitter.QueryCapture, returnValue T) (T, error)

func WithMatches[T any](query string, language string, content []byte, returnValue T, handler HandleMatch[T]) (T, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(GetLanguage(language))

	tree, err := parser.ParseCtx(context.TODO(), nil, content)
	if err != nil {
		return returnValue, err
	}

	qc, q, err := GetQuery(query, language)
	if err != nil {
		return returnValue, err
	}

	qc.Exec(q, tree.RootNode())

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		m = qc.FilterPredicates(m, content)

		returnValue, err = handler(m.Captures, returnValue)
		if err != nil {
			return returnValue, err
		}
	}

	return returnValue, nil
}
