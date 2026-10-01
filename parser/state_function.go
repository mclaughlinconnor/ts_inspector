package parser

import (
	"ts_inspector/ast/manipulation"
)

// Only used for interesting points

type Function struct {
	BodyNode       manipulation.AstManipulationNode // potentially missing
	NameNode       *manipulation.Identifier
	Node           manipulation.FunctionSignature
	ParametersNode *manipulation.FormalParameters
}
