package utils

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Captures = map[string][]*sitter.Node

type HandleMatch[T any] func(captures Captures, returnValue T) (T, error)

func WithMatches[T any](query string, language string, content []byte, returnValue T, handler HandleMatch[T]) (T, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(GetLanguage(language))

	tree := parser.Parse(content, nil)

	qc, q, err := GetQuery(query, language)
	if err != nil {
		return returnValue, err
	}

	matches := qc.Matches(q, tree.RootNode(), content)

	for {
		m := matches.Next()
		if m == nil {
			break
		}

		captures := map[string][]*sitter.Node{}
		captureNames := q.CaptureNames()

		for _, capture := range m.Captures {
			if captures[captureNames[capture.Index]] != nil {
				captures[captureNames[capture.Index]] = append(captures[captureNames[capture.Index]], &capture.Node)
			} else {
				captures[captureNames[capture.Index]] = []*sitter.Node{&capture.Node}
			}
		}

		returnValue, err = handler(captures, returnValue)
		if err != nil {
			return returnValue, err
		}
	}

	return returnValue, nil
}
