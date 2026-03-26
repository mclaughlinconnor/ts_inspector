package tcb_port

type TcbExpressionOp struct {
	TcbOp
	tcb         *Context
	scope       *Scope
	expression  *Expression
	isBoundText bool
}

func (t TcbExpressionOp) Optional() bool            { return false }
func (t TcbExpressionOp) CircularFallback() TcbExpr { return TcbExpr{Source: "null!"} }

func (t TcbExpressionOp) Execute() *Identifier {
	if t.expression == nil {
		return nil
	}

	expr := AstToTypescript(t.expression)

	statement := Statement{}
	if t.isBoundText {
		statement.AddPart("\"\" + (")
		statement.AddPart(expr)
		statement.AddPart(");")
	} else {
		statement.AddPart("(")
		statement.AddPart(expr)
		statement.AddPart(");")
	}

	return nil
}
