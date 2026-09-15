package manipulation

import (
	"slices"
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type action struct {
	Name    string
	Perform func() []utils.TextEdit
}

type editSession struct {
	afterDocument  string
	beforeDocument string
}

type element struct {
	endOffset   uint
	startOffset uint
}

type elementEdit struct {
	newEndOffset uint
	oldEndOffset uint
	startOffset  uint
}

type programContent struct {
	editSession *editSession
	root        *root
	text        []byte
	tree        *sitter.Tree
}

func (e *editSession) buildLspTextEdit() utils.TextEdit {
	editStart := 0

	afterDocument := e.afterDocument
	beforeDocument := e.beforeDocument

	lenAfterDocument := len(afterDocument)
	lenBeforeDocument := len(beforeDocument)

	for i := range afterDocument {
		if i >= lenAfterDocument || i >= lenBeforeDocument {
			break
		}

		if afterDocument[i] == beforeDocument[i] {
			continue
		}

		editStart = i
		break
	}

	afterEditEnd := lenAfterDocument - 1
	beforeEditEnd := lenBeforeDocument - 1

	for afterEditEnd >= 0 && beforeEditEnd >= 0 {
		if afterDocument[afterEditEnd] == beforeDocument[beforeEditEnd] {
			afterEditEnd--
			beforeEditEnd--
			continue
		}

		afterEditEnd = min(afterEditEnd+1, lenAfterDocument-1)
		beforeEditEnd = min(beforeEditEnd+1, lenBeforeDocument-1)

		break
	}

	startPosition := utils.GetPositionForOffset2(beforeDocument, editStart)
	endPosition := utils.GetPositionForOffset2(beforeDocument, beforeEditEnd)

	r := utils.Range{Start: startPosition, End: endPosition}
	newText := afterDocument[editStart:afterEditEnd]

	return utils.TextEdit{Range: r, NewText: newText}
}

func (e *element) edit(edit *elementEdit) {
	if e.getStartOffset() < edit.startOffset && e.getEndOffset() < edit.startOffset {
		return
	}

	difference := edit.newEndOffset - edit.oldEndOffset
	if e.getStartOffset() >= edit.startOffset {
		e.startOffset += difference
	}

	e.endOffset += difference
}

func (e *element) getEndOffset() uint {
	return e.endOffset
}

func (e *element) getStartOffset() uint {
	return e.startOffset
}

func (p *programContent) beginEditSession() {
	p.editSession = &editSession{beforeDocument: string(p.text)}
}

func (p *programContent) editText(startOffset uint, endOffset uint, replacementText string) {
	p.text = slices.Replace(p.text, int(startOffset), int(endOffset), []byte(replacementText)...)
	p.updateEditSessionSnapshot(string(p.text))
}

func (p *programContent) editTree(edit *elementEdit) {
	p.root.visit(func(ni nodeInterface) int {
		ni.getElement().edit(edit)
		return VisitContinue
	})
}

func (p *programContent) endEditSession() {
	p.editSession = nil
}

func (p *programContent) updateEditSessionSnapshot(document string) {
	p.editSession.afterDocument = document
}

func elementFromNode(node *sitter.Node) *element {
	return &element{endOffset: node.EndByte(), startOffset: node.StartByte()}
}
