package tcb

type TcbBlockVariableOp struct {
	TcbOp
	tcb         *Context
	scope       *Scope
	initializer Expression
	variable    TmplAstVariable
}

func (o TcbBlockVariableOp) Optional() bool { return false }
func (o TcbBlockVariableOp) Execute() Identifier {
	id := o.tcb.allocateId(o.variable, nil)

	variable := tsCreateVariable(id, o.initializer, false)
	o.scope.addStatementStatement(variable)

	return id
}

/**
 * A `TcbOp` which renders a variable that is implicitly available within a block (e.g. `$count`
 * in a `@for` block).
 *
 * Executing this operation returns the identifier which can be used to refer to the variable.
 */

type TcbBlockImplicitVariableOp struct {
	TcbOp
	tcb         *Context
	scope       *Scope
	ttype       Expression
	variable    *TmplAstVariable
	initializer *Expression
}

func (o TcbBlockImplicitVariableOp) Optional() bool { return true }
func (o TcbBlockImplicitVariableOp) Execute() Identifier {
	id := o.tcb.allocateId(o.variable, nil)

	variable := tsDeclareVariable(id, o.ttype, o.initializer)
	o.scope.addStatementStatement(variable)

	return id
}
