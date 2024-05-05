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
	QueryPrototypeUsage     = "query_prototype_usage"
	QueryPropertyDefinition = "query_property_definition"
	QueryMethodDefinition   = "query_method_definition"
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

var typescriptPrototypeUsage = []byte(`
  [
    (member_expression
      object: (member_expression
        object: (identifier) @class
        property: (property_identifier) @prototype)
      property: (property_identifier) @var)
    (subscript_expression
      object: (member_expression
        object: (identifier) @class
        property: (property_identifier) @prototype)
      index: (string
        (string_fragment) @var))
    (#eq? @prototype "prototype")
    ; (#eq? @class "class") ; add later when class checking is supported
  ]
`)

var typescriptPropetyDefinition = []byte(`
  [
    (public_field_definition
      decorator: [
        (decorator
          (call_expression
            function: (identifier) @decorator))
        (decorator (identifier) @decorator)
      ]*
      (accessibility_modifier) @accessibility_modifier
      name: (property_identifier) @var) @definition
    (required_parameter
      decorator: (decorator
        (call_expression
          function: (identifier) @decorator))*
      (accessibility_modifier) @accessibility_modifier
      pattern: (identifier) @var) @definition
  ]
`)

var typescriptMethodDefinition = []byte(`
  (
    [
      (decorator
        (call_expression
          function: (identifier) @decorator))
      (decorator (identifier) @decorator)
    ]*
    .
    (method_definition
      (accessibility_modifier)? @accessibility_modifier
      "static"? @static
      (override_modifier)? @override
      "readonly"? @readonly
      "async"? @async
      "get"? @get
      name: (property_identifier) @name
      ; "?" ; Unhandled
    )
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
	registerQuery(QueryPropertyUsage, TypeScript, typescriptPropertyUsage)
	registerQuery(QueryPropertyUsage, JavaScript, javascriptPropertyUsage)
	registerQuery(QueryContent, Pug, pugContent)
	registerQuery(QueryAttribute, Pug, pugAttribute)
	registerQuery(QueryInterpolation, AngularContent, angularContentInterpolation)
	registerQuery(QueryPrototypeUsage, TypeScript, typescriptPrototypeUsage)
	registerQuery(QueryPropertyDefinition, TypeScript, typescriptPropetyDefinition)
	registerQuery(QueryMethodDefinition, TypeScript, typescriptMethodDefinition)
}
