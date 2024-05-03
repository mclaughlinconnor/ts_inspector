package main

import (
	"log"

	sitter "github.com/smacker/go-tree-sitter"
)

var queries = map[string]map[string][]byte{}

const (
	QueryComponentDecorator = "query_component_decorator"
)

var componentDecorator = []byte(`
  (decorator
    (call_expression
      function: (identifier) @decorator_name
      arguments: (arguments
        (object
          (pair
             key: (property_identifier) @key_name
             value: (string (string_fragment) @template)
          )
        )
      )
    )
    (#eq? @key_name "templateUrl")
    (#eq? @decorator_name "Component")
  )
  `)

func registerQuery(name string, lang string, query []byte) {
	_, ok := queries[lang]
	if !ok {
		queries[lang] = make(map[string][]byte, 0)
	}

	queries[lang][name] = query
}

func GetQuery(name string, lang string) (*sitter.QueryCursor, *sitter.Query) {
	q, ok := queries[lang][name]
	if !ok {
		log.Fatalf("No query '%s' found", name)
	}
	query, err := sitter.NewQuery(q, GetLanguage(lang))
	if err != nil {
		log.Fatal(err)
	}

	return sitter.NewQueryCursor(), query
}

func InitQueries() {
	registerQuery(QueryComponentDecorator, TypeScript, componentDecorator)
}
