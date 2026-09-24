package manipulation

import "ts_inspector/interfaces"

type Ast struct {
	commonNode
	Program nodeInterface
}

func (a *Ast) GetCfg() (*Cfg, error) {
	var outsideError error

	a.visit(func(ni nodeInterface) int {
		built, err := ni.buildCfgBlock()
		if err != nil {
			outsideError = err
			return VisitAbort
		}

		if built {
			return VisitSkip
		}

		return VisitContinue
	})

	return a.getCfg(), outsideError
}

func (a *Ast) GetAllActions(offset uint) []action {
	actions := []action{}
	a.visit(func(ni nodeInterface) int {
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

func (a *Ast) GetAllAnalysis() []interfaces.Analysis {
	analyses := []interfaces.Analysis{}
	a.visit(func(ni nodeInterface) int {
		as := ni.getAnalysis()
		if len(as) != 0 {
			analyses = append(analyses, as...)
		}

		return VisitContinue
	})

	return analyses
}

func (a *Ast) getAstNodeOfKindAtOffset(offset uint, kind string, first bool) (nodeInterface, bool) {
	if kind == NULL_KIND || a.getKind() == kind {
		return a, true
	}

	return a.Program.getAstNodeOfKindAtOffset(offset, kind, first)
}

func (a *Ast) visit(exec func(nodeInterface) int) int {
	if ret := exec(a); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	return a.Program.visit(exec)
}
