package parser

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func FileExists(filename string) bool {
	stat, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !stat.IsDir()
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

func FilenameFromUri(uri string) string {
	return strings.TrimPrefix(uri, `file://`)
}

func UriFromFilename(filename string) string {
	return `file://` + filename
}

func CStr2GoStr(b []byte) string {
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		i = len(b)
	}

	return string(b[:i])
}

func GetLogger(filename string) *log.Logger {
	filename = "/home/connor/Development/ts_inspector/" + filename
	logfile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic(fmt.Sprintf("Invalid file: %s", filename))
	}

	return log.New(logfile, "[ts_inspector]", log.Ldate|log.Ltime|log.Lshortfile)
}
