package tcb_port

import (
	"fmt"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

/**
 * A `TcbOp` which generates code for checking input bindings on a directive.
 *
 * Executing this operation returns a reference to the directive instance variable with its
 * inputs properly assigned.
 */
type TcbDirectiveInputsOp struct {
	tcb                *Context
	scope              *Scope
	node               *TmplAstNode
	dir                *TmplDirectiveMetadata
	isDynamicDirective bool
	class              *parser.Class
}

func (o TcbDirectiveInputsOp) Optional() bool            { return false }
func (o TcbDirectiveInputsOp) CircularFallback() TcbExpr { return TcbExpr{Source: "null!"} }

func (o TcbDirectiveInputsOp) Execute() *Identifier {
	inputs := o.class.GetInputs()
	if len(inputs) == 0 {
		return nil
	}

	// // Build a map of input binding name -> field name
	// // inputName is the public name exposed in templates (may differ from field name via @Input('alias'))
	// inputBindings := make(map[string]string) // binding name -> field name
	// for _, input := range inputs {
	// 	bindingName := input.GetInputName() // public name (respects @Input('alias'))
	// 	fieldName := input.Name             // actual class property name
	// 	inputBindings[bindingName] = fieldName
	// }
	//
	// Resolve the directive instance - we need it to assign inputs to
	var dirId *Identifier

	// Iterate over the node's attributes and match them to directive inputs
	for _, attrNode := range o.node.Tag.Attributes.elems {
		if attrNode.Attribute == nil {
			continue
		}

		attr := attrNode.Attribute

		// Strip Angular binding syntax from attribute name
		// e.g. "[propertyName]" -> "propertyName", "[(ngModel)]" -> "ngModel"
		strippedName, mode := utils.StripAngularFromAttribute(attr.Name)
		if mode == utils.OutputAngularStripped {
			// This is an output binding (event), skip it for inputs
			continue
		}

		// Check if this attribute matches any of the directive's inputs
		fieldName, found := inputBindings[strippedName]
		if !found {
			// Also check against the raw attribute name (for non-binding attributes)
			fieldName, found = inputBindings[attr.Name]
		}
		if !found {
			continue
		}

		// Lazily resolve the directive instance
		if dirId == nil {
			resolved := o.scope.resolve(*o.node, o.dir)
			dirId = resolved
		}

		// Translate the attribute value to a TypeScript expression
		var exprStr string
		if attr.Value != "" {
			if (mode & utils.InputAngularStripped) > 0 {
				// This is a property binding [prop]="expr" — the value is an expression
				expr := &Expression{attr.Value}
				exprStr = AstToTypescript(expr)
			} else {
				// This is a static attribute prop="value" — the value is a string literal
				exprStr = fmt.Sprintf("%q", attr.Value)
			}
		} else {
			exprStr = "undefined"
		}

		// Generate: dirInstance.fieldName = expression;
		target := fmt.Sprintf("%s.%s", IdentifierName(*dirId), fieldName)
		if !isJavascriptIdentifier(fieldName) {
			target = fmt.Sprintf("%s[\"%s\"]", IdentifierName(*dirId), fieldName)
		}

		statement := Statement{}
		statement.AddPart(target)
		statement.AddPart(" = ")
		statement.AddPart(exprStr)
		statement.AddPart(";")

		o.scope.addStatementStatement(statement)
	}
	//
	// return nil
}
