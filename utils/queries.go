package utils

import (
	"fmt"

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
	QueryClassDefinition    = "query_class_implements"
	QueryImport             = "query_imports"
	QueryClassProperties    = "query_class_properties"
	QueryClassBody          = "query_class_body"
)

var typescriptComponentDecorator = []byte(`
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

var typescriptPropertyDefinition = []byte(`
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
      "*"? @generator
      name: (property_identifier) @name
      ; "?" ; Unhandled
    ) @definition
  )
`)

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

var typescriptClassProperties = []byte(`
  (class_body
    ((comment) @comment)*
    [
      ((public_field_definition
        name: [(property_identifier) (private_property_identifier)] @name) @node (";" @semi)?)
      ((method_definition
        name: [(property_identifier) (private_property_identifier)] @name) @node)
      ((method_signature
        name: [(property_identifier) (private_property_identifier)] @name) @node (";" @semi)?)
      ((abstract_method_signature
        name: [(property_identifier) (private_property_identifier)] @name) @node)
    ]
  )
`)

var typescriptClassBody = []byte(`(class_body) @body`)

func registerQuery(name string, lang string, query []byte) {
	_, ok := queries[lang]
	if !ok {
		queries[lang] = make(map[string][]byte, 0)
	}

	queries[lang][name] = query
}

func GetQuery(name string, lang string) (*sitter.QueryCursor, *sitter.Query, error) {
	q, ok := queries[lang][name]
	if !ok {
		return nil, nil, fmt.Errorf("No query for '%s' found", name)
	}
	query, err := sitter.NewQuery(q, GetLanguage(lang))
	if err != nil {
		return nil, nil, err
	}

	return sitter.NewQueryCursor(), query, nil
}

func InitQueries() {
	registerQuery(QueryComponentDecorator, TypeScript, typescriptComponentDecorator)
	registerQuery(QueryPropertyUsage, TypeScript, typescriptPropertyUsage)
	registerQuery(QueryPropertyUsage, JavaScript, javascriptPropertyUsage)
	registerQuery(QueryContent, Pug, pugContent)
	registerQuery(QueryAttribute, Pug, pugAttribute)
	registerQuery(QueryInterpolation, AngularContent, angularContentInterpolation)
	registerQuery(QueryPrototypeUsage, TypeScript, typescriptPrototypeUsage)
	registerQuery(QueryPropertyDefinition, TypeScript, typescriptPropertyDefinition)
	registerQuery(QueryMethodDefinition, TypeScript, typescriptMethodDefinition)
	registerQuery(QueryClassDefinition, TypeScript, typescriptClassDefinition)
	registerQuery(QueryImport, TypeScript, typescriptImport)
	registerQuery(QueryClassProperties, TypeScript, typescriptClassProperties)
	registerQuery(QueryClassBody, TypeScript, typescriptClassBody)
}
