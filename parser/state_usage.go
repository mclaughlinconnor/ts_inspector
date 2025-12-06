package parser

import sitter "github.com/smacker/go-tree-sitter"

type Usage struct {
	Access access
	Name   string
	Usages []*UsageInstance
}

type Usages map[string]Usage

type UsageInstance struct {
	Access access
	Class  *Class
	Node   *sitter.Node
}

type access struct {
	Modifier   string
	Precedence int
}

var NoAccess = access{"none", 0}
var ConstructorAccess = access{"constructor", 1}
var LocalAccess = access{"local", 2}
var TemplateAccess = access{"template", 3}
