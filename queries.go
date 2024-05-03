package main

import (
	"log"

	sitter "github.com/smacker/go-tree-sitter"
)

var queries = map[string]map[string][]byte{}

const (
	QueryComponentDecorator = "query_component_decorator"
	QueryPropertyUsage      = "query_property_usage"
	QueryContent            = "query_content"
	QueryAttribute          = "query_attribute"
	QueryInterpolation      = "query_interpolation"
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
    (#eq? @decorator_name "Component"))
`)

var typescriptPropertyUsage = []byte(`
  (member_expression
    object: (this)
    property: (property_identifier) @var)
`)

var javascriptPropertyUsage = []byte(`
  (identifier) @name
`)

var pugAttribute = []byte(`
  (attribute
    (attribute_name) @name
    (quoted_attribute_value
      (attribute_value) @value))
`)

var pugContent = []byte(`
  (content) @content
`)

var angularContentInterpolation = []byte(`
  (interpolation) @interpolation
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
	registerQuery(QueryPropertyUsage, TypeScript, typescriptPropertyUsage)
	registerQuery(QueryPropertyUsage, JavaScript, javascriptPropertyUsage)
	registerQuery(QueryContent, Pug, pugContent)
	registerQuery(QueryAttribute, Pug, pugAttribute)
	registerQuery(QueryInterpolation, AngularContent, angularContentInterpolation)
}
