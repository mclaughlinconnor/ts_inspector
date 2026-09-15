package manipulation

type boolean struct {
	commonNode
}

func (b *boolean) getAstNodeAtOffset(offset uint) (nodeInterface, bool) {
	return b.getAstNodeOfKindAtOffset(offset, NULL_KIND, false)
}

func (b *boolean) getAstNodeOfKindAtOffset(offset uint, kind string, _ bool) (nodeInterface, bool) {
	if kind != NULL_KIND && b.getKind() != kind {
		return nil, false
	}

	node := b.getTsNode()
	if node.StartByte() <= offset && offset < node.EndByte() {
		return b, true
	}

	return nil, false
}

func (b *boolean) getValue() bool {
	node := b.getTsNode()

	return node.Kind() == "true"
}

func (b *boolean) invert() {
	if b.getValue() {
		b.editText("false")
	} else {
		b.editText("true")
	}
}

func (b *boolean) visit(exec func(nodeInterface) int) int {
	if ret := exec(b); ret != VisitContinue {
		if ret == VisitAbort {
			return ret
		}

		if ret == VisitSkip {
			return VisitContinue
		}
	}

	return VisitContinue
}
