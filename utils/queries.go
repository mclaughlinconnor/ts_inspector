package utils

import (
	"fmt"
	"log"

	sitter "github.com/smacker/go-tree-sitter"
)

var queries = map[string]map[string]*sitter.Query{}

const (
	QueryClassDefinition = "query_class_implements"
	QueryImport          = "query_imports"
	QueryClassBody       = "query_class_body"
)

var typescriptClassDefinition = []byte(`
  (class_declaration
    name: (type_identifier) @name
    type_parameters: (type_parameters)? @type_parameters
    (class_heritage
      (extends_clause)? @extends_clause
      (implements_clause
        (type_identifier) @identifier)? @implements_clause)?)
`)

var typescriptImport = []byte(`
  (import_statement
    "type"? @type
    (import_clause
      (named_imports) @named_imports) @clause
    source: (string
      (string_fragment) @package)) @import
`)

var typescriptClassBody = []byte(`(class_body) @body`)

func registerQuery(name string, lang string, queryString []byte) {
	_, ok := queries[lang]
	if !ok {
		queries[lang] = make(map[string]*sitter.Query, 0)
	}

	query, err := sitter.NewQuery(queryString, GetLanguage(lang))
	if err != nil {
		log.Fatal(err)
	}

	queries[lang][name] = query
}

func GetQuery(name string, lang string) (*sitter.QueryCursor, *sitter.Query, error) {
	query, ok := queries[lang][name]
	if !ok {
		return nil, nil, fmt.Errorf("No query for '%s' found", name)
	}

	return sitter.NewQueryCursor(), query, nil
}

func InitQueries() {
	registerQuery(QueryClassDefinition, TypeScript, typescriptClassDefinition)
	registerQuery(QueryImport, TypeScript, typescriptImport)
	registerQuery(QueryClassBody, TypeScript, typescriptClassBody)
}
