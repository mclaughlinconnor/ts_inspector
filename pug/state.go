package pug

import sitter "github.com/smacker/go-tree-sitter"

type NodeType struct {
	name string
}

var ATTRIBUTE = NodeType{"ATTRIBUTE"}
var ATTRIBUTE_NAME = NodeType{"ATTRIBUTE_NAME"}
var CONTENT = NodeType{"CONTENT"}
var EMPTY = NodeType{"EMPTY"}
var EQUALS = NodeType{"EQUALS"}
var FILENAME = NodeType{"FILENAME"}
var ID_CLASS = NodeType{"ID_CLASS"}
var JAVASCRIPT = NodeType{"JAVASCRIPT"}
var SPACE = NodeType{"SPACE"}
var TAG = NodeType{"TAG"}
var TAG_NAME = NodeType{"TAG_NAME"}

type State struct {
	HtmlText string
	PugText  string
	Ranges   []Range
}

type Range struct {
	HtmlEnd   uint32
	HtmlStart uint32
	PugEnd    uint32
	PugStart  uint32
	NodeType  NodeType
}

type NodeRange struct {
	EndIndex      uint32
	StartIndex    uint32
	StartPosition sitter.Point
	EndPosition   sitter.Point
}
