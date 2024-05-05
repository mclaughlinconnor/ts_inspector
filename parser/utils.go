package parser

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

type HandleMatch[T any] func(captures []sitter.QueryCapture, returnValue T) (T, error)

func WithMatches[T any](query string, language string, content []byte, returnValue T, handler HandleMatch[T]) (T, error) {
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

func IsAngularDecorator(name string) bool {
	_, found := angularDecorators[name]

	return found
}

var angularDecorators = map[string]bool{
	"Attribute":       true,
	"Component":       true,
	"ContentChild":    true,
	"ContentChildren": true,
	"Directive":       true,
	"Host":            true,
	"HostBinding":     true,
	"HostListener":    true,
	"Inject":          true,
	"Injectable":      true,
	"Input":           true,
	"NgModule":        true,
	"Optional":        true,
	"Output":          true,
	"Pipe":            true,
	"Self":            true,
	"SkipSelf":        true,
	"ViewChild":       true,
	"ViewChildren":    true,
}
