package main

import (
	"context"
	"log"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
)

func FileExists(filename string) bool {
	stat, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}

func ReadFile(filename string) []byte {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	data := make([]byte, 10240)

	_, err = file.Read(data)
	if err != nil {
		log.Fatal(err)
	}

	return data
}

type HandleCapture[T any] func(captures []sitter.QueryCapture, returnValue T) (T, error)

func WithCaptures[T any](query string, language string, content []byte, handler HandleCapture[T]) (T, error) {
	var returnValue T

	parser := sitter.NewParser()
	parser.SetLanguage(GetLanguage(language))

	tree, err := parser.ParseCtx(context.TODO(), nil, content)
	if err != nil {
		log.Fatal(err)
	}

	qc, q := GetQuery(query, language)
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
