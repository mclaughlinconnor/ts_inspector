package manipulation

type root struct {
	commonNode
	Program nodeInterface
}

func (r *root) GetAllActions(offset uint) []action {
	actions := []action{}
	r.visit(func(ni nodeInterface) int {
		if !ni.isUnderCursor(offset) {
			return VisitSkip
		}

		as := ni.getActions()
		if len(as) != 0 {
			actions = append(actions, as...)
		}

		return VisitContinue
	})

	return actions
}

func (r *root) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if kind == NULL_KIND || r.getKind() == kind {
		return r, true
	}

	return r.Program.getAstNodeOfKindAtOffset(offset, kind, first)
}

func (r *root) visit(exec func(nodeInterface) int) int {
	if ret := exec(r); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	return r.Program.visit(exec)
}

