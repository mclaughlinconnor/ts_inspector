package main

import sitter "github.com/smacker/go-tree-sitter"

type access struct {
	modifier   string
	precedence int
}

var ConstructorAccess = access{"constructor", 0}
var LocalAccess = access{"local", 1}
var ForeignAccess = access{"foreign", 2}

type accessibility struct {
	Modifier string
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

func CalculateNewAccessType(new access, old access) access {
	if new.precedence > old.precedence {
		return new
	}

	return old
}

type State struct {
	Usages Usages
}

func NewState() State {
	return State{Usages{}}
}
