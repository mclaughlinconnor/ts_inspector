package utils

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"
)

type Captures = map[string][]*sitter.Node

type HandleMatch[T any] func(captures Captures, returnValue T) (T, error)

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

		captures := map[string][]*sitter.Node{}
		for _, capture := range m.Captures {
			if captures[q.CaptureNameForId(capture.Index)] != nil {
				captures[q.CaptureNameForId(capture.Index)] = append(captures[q.CaptureNameForId(capture.Index)], capture.Node)
			} else {
				captures[q.CaptureNameForId(capture.Index)] = []*sitter.Node{capture.Node}
			}
		}

		returnValue, err = handler(captures, returnValue)
		if err != nil {
			return returnValue, err
		}
	}

	return returnValue, nil
}
