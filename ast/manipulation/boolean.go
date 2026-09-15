package manipulation

type boolean struct {
	commonNode
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
