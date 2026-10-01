package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type accessibilityModifierKind = string

//nolint:unused
type accessibilityModifierKindStruct struct {
	PRIVATE   string
	PROTECTED string
	PUBLIC    string
}

//nolint:unused
var accessibilityModifierKindEnum = accessibilityModifierKindStruct{"public", "private", "protected"}

type accessibilityModifier struct {
	commonNode
}

func (b *accessibilityModifier) getKind() accessibilityModifierKind {
	return b.getText()
}

//nolint:unused
func (b *accessibilityModifier) isPrivate() bool {
	return b.getText() == accessibilityModifierKindEnum.PRIVATE
}

//nolint:unused
func (b *accessibilityModifier) isProtected() bool {
	return b.getText() == accessibilityModifierKindEnum.PROTECTED
}

//nolint:unused
func (b *accessibilityModifier) isPublic() bool {
	return b.getText() == accessibilityModifierKindEnum.PUBLIC
}

func visitAccessibilityModifier(node *sitter.Node, state walkState, indexInParent uint, funcMap walk.VisitorFuncMap[walkState]) (walkState, error) {
	accessibilityModifier := accessibilityModifier{commonNode: makeCommonNode("accessibilityModifier", state, node)}
	accessibilityModifier.setImpl(&accessibilityModifier)

	return &accessibilityModifier, nil
}
