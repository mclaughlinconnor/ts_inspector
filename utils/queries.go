package utils

import (
	"fmt"
	"log"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

var queries = map[string]map[string]*sitter.Query{}

const (
	QueryClassDefinition = "query_class_implements"
	QueryImport          = "query_imports"
	QueryClassBody       = "query_class_body"
)

var typescriptClassDefinition = `
  (class_declaration
    name: (type_identifier) @name
    type_parameters: (type_parameters)? @type_parameters
    (class_heritage
      (extends_clause)? @extends_clause
      (implements_clause
        (type_identifier) @identifier)? @implements_clause)?)
`

var typescriptImport = `
  (import_statement
    "type"? @type
    (import_clause
      (named_imports) @named_imports) @clause
    source: (string
      (string_fragment) @package)) @import
`

var typescriptClassBody = `(class_body) @body`

func registerQuery(name string, lang string, queryString string) {
	_, ok := queries[lang]
	if !ok {
		queries[lang] = make(map[string]*sitter.Query, 0)
	}

	query, err := sitter.NewQuery(GetLanguage(lang), queryString)
	if err != nil {
		log.Fatal(err)
	}

	queries[lang][name] = query
}

func GetQuery(name string, lang string) (*sitter.QueryCursor, *sitter.Query, error) {
	query, ok := queries[lang][name]
	if !ok {
		return nil, nil, fmt.Errorf("no query for '%s' found", name)
	}

	return sitter.NewQueryCursor(), query, nil
}

func InitQueries() {
	registerQuery(QueryClassDefinition, TypeScript, typescriptClassDefinition)
	registerQuery(QueryImport, TypeScript, typescriptImport)
	registerQuery(QueryClassBody, TypeScript, typescriptClassBody)
}
