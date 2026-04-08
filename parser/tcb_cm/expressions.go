package tcb_cm

import (
	"slices"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type Expression struct {
	Parts []StatementParts
}

type Part struct {
	PugEndOffset   *int
	PugStartOffset *int
	TsEndOffset    *int
	TsStartOffset  *int

	node *sitter.Node
	text string
}

func (p *Part) IsReal() bool {
	return p.node != nil
}

func (p *Part) IsVirtual() bool {
	return p.node == nil
}

type StatementParts struct {
	Parts  []Part
	sb     strings.Builder
	length int
}

func (s *StatementParts) AddCodeBlock(builder func()) {
	s.AddVirtPart("{\n")
	builder()
	s.AddVirtPart("\n}")
}

func (s *StatementParts) AddVirtPart(text string) {
	s.AddRealPart(text, nil)
}

func (s *StatementParts) AddStatementParts(statementParts *StatementParts) {
	for _, p := range statementParts.Parts {
		tsStartOffset := s.sb.Len()
		tsEndOffset := tsStartOffset + (*p.TsEndOffset - *p.TsStartOffset)

		newPart := Part{
			node:           p.node,
			text:           p.text,
			TsStartOffset:  &tsStartOffset,
			TsEndOffset:    &tsEndOffset,
			PugStartOffset: p.PugStartOffset,
			PugEndOffset:   p.PugEndOffset,
		}

		s.AddPart(newPart)
	}
}

func (s *StatementParts) OffsetByNodeStart(node *sitter.Node) {
	offset := int(node.StartByte())

	for _, p := range s.Parts {
		(*p.PugStartOffset) += offset
		(*p.PugEndOffset) += offset
	}
}

func (s *StatementParts) PrependVirtPart(text string) {
	tsStartOffset := 0
	tsEndOffset := len(text)

	p := Part{text: text, TsStartOffset: &tsStartOffset, TsEndOffset: &tsEndOffset}
	s.Parts = slices.Insert(s.Parts, 0, p)

	for _, p := range s.Parts {
		(*p.TsStartOffset) += tsStartOffset
		(*p.TsEndOffset) += tsEndOffset
	}
}

func (s *StatementParts) AddPart(part Part) {
	if part.node != nil && (part.PugStartOffset == nil || part.PugEndOffset == nil) {
		start := int(part.node.StartByte())
		end := int(part.node.EndByte())

		part.PugStartOffset = &start
		part.PugEndOffset = &end
	}

	s.Parts = append(s.Parts, part)
	s.sb.WriteString(part.text)
}

func (s *StatementParts) AddRealPart(text string, node *sitter.Node) {
	tsStartOffset := s.sb.Len()
	tsEndOffset := tsStartOffset + len(text)

	s.AddPart(Part{node: node, text: text, TsEndOffset: &tsEndOffset, TsStartOffset: &tsStartOffset})
}

func (s *StatementParts) AppendStatement(statement StatementParts) {
	s.AddStatementParts(&statement)
	s.AddVirtPart("\n")
}

func (s *StatementParts) ToString() string {
	sb := strings.Builder{}
	for _, p := range s.Parts {
		sb.WriteString(p.text)
	}

	return sb.String()
}

func StatementPartsFromNodeContent(node *sitter.Node, content []byte) *StatementParts {
	nodeContent := node.Content(content)

	statementParts := StatementParts{}
	statementParts.AddRealPart(nodeContent, node)

	return &statementParts
}
