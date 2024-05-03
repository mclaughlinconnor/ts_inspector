package main

import sitter "github.com/smacker/go-tree-sitter"

type access struct {
	modifier string
}

var LocalAccess = access{"local"}
var ForeignAccess = access{"foreign"}

type accessibility struct {
	modifier string
}

var PublicAccessibility = accessibility{"public"}
var PrivateAccessibility = accessibility{"private"}

type Identifier struct {
	AccessModifier accessibility
	Name           string
	Node           *sitter.Node
}

type UsageInstance struct {
	Access access
	Node   *sitter.Node
}

type Usage struct {
	Access access
	Name   string
	Usages []UsageInstance
}

type Usages map[string]Usage
