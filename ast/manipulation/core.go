package manipulation

import (
	"fmt"
	"slices"
	"ts_inspector/utils"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type action struct {
	Name    string
	Perform func() ([]utils.TextEdit, error)
}

type editSession struct {
	afterDocument  []byte
	beforeDocument []byte
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

	beforeDocumentString := string(beforeDocument)
	afterDocumentString := string(afterDocument)

	startPosition := utils.GetPositionForOffset2(beforeDocumentString, editStart)
	endPosition := utils.GetPositionForOffset2(beforeDocumentString, beforeEditEnd)

	r := utils.Range{Start: startPosition, End: endPosition}
	newText := afterDocumentString[editStart:afterEditEnd]

	return utils.TextEdit{Range: r, NewText: newText}
}

func (e *element) copy() *element {
	return &element{endOffset: e.getEndOffset(), startOffset: e.getStartOffset()}
}

func (e *element) edit(edit *elementEdit) {
	if e.getStartOffset() < edit.startOffset && e.getEndOffset() < edit.startOffset {
		return
	}

	difference := edit.newEndOffset - edit.oldEndOffset
	if e.getStartOffset() > edit.startOffset {
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

func (p *programContent) beginEditSession() error {
	if p.hasEditSession() {
		return fmt.Errorf("tried to start an edit session when there is already an edit session in progress")
	}

	p.editSession = &editSession{afterDocument: slices.Clone(p.text), beforeDocument: slices.Clone(p.text)}

	var err error = nil

	p.root.visit(func(ni nodeInterface) int {
		err = ni.beginEditSession()
		if err != nil {
			return VisitAbort
		}

		return VisitContinue
	})

	return err
}

func (p *programContent) commitEditSession() error {
	if p.hasEditSession() {
		return fmt.Errorf("tried to commit an edit session when there is no edit session in progress")
	}

	p.text = p.editSession.afterDocument
	p.editSession = nil

	var err error = nil

	p.root.visit(func(ni nodeInterface) int {
		err = ni.dropEditSession()
		if err != nil {
			return VisitAbort
		}

		return VisitContinue
	})

	return err
}

func (p *programContent) dropEditSession() error {
	if !p.hasEditSession() {
		return fmt.Errorf("tried to drop an edit session when there is no edit session in progress")
	}

	p.text = []byte(p.editSession.beforeDocument)
	p.editSession = nil

	var err error = nil

	p.root.visit(func(ni nodeInterface) int {
		err = ni.dropEditSession()
		if err != nil {
			return VisitAbort
		}

		return VisitContinue
	})

	return err
}

func (p *programContent) editText(startOffset uint, endOffset uint, replacementText string) {
	p.text = slices.Replace(p.text, int(startOffset), int(endOffset), []byte(replacementText)...)
	p.updateEditSessionSnapshot(p.text)
}

func (p *programContent) editTree(edit *elementEdit) {
	p.root.visit(func(ni nodeInterface) int {
		ni.getElement().edit(edit)
		return VisitContinue
	})
}

func (p *programContent) getText() string {
	if p.editSession != nil {
		return string(p.editSession.afterDocument)
	}

	return string(p.text)
}

func (p *programContent) getRoot() *root {
	return p.root
}

func (p *programContent) hasEditSession() bool {
	return p.editSession != nil
}

func (p *programContent) updateEditSessionSnapshot(document []byte) {
	p.editSession.afterDocument = slices.Clone(document)
}

func elementFromNode(node *sitter.Node) *element {
	return &element{endOffset: node.EndByte(), startOffset: node.StartByte()}
}
