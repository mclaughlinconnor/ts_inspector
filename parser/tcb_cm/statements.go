package tcb_cm

import (
	"slices"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type Part struct {
	PugEndOffset   *int
	PugStartOffset *int
	TsEndOffset    *int
	TsStartOffset  *int

	node  *sitter.Node
	text  string
	scope *Scope
}

func (p *Part) IsReal() bool {
	return p.node != nil
}

func (p *Part) IsVirtual() bool {
	return p.node == nil
}

type Statement struct {
	Parts []Part
	sb    strings.Builder
}

func (s *Statement) AddCodeBlock(builder func()) {
	s.AddVirtPart("{\n")
	builder()
	s.AddVirtPart("\n}")
}

func (s *Statement) AddVirtPart(text string) {
	s.AddRealPart(text, nil)
}

func (s *Statement) AddStatement(statement *Statement) {
	for _, p := range statement.Parts {
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

func (s *Statement) OffsetByNodeStart(node *sitter.Node) {
	offset := int(node.StartByte())

	for _, p := range s.Parts {
		if p.node == nil || p.PugStartOffset == nil || p.PugEndOffset == nil {
			continue
		}

		(*p.PugStartOffset) += offset
		(*p.PugEndOffset) += offset
	}
}

func (s *Statement) PrependVirtPart(text string) {
	tsStartOffset := 0
	tsEndOffset := len(text)

	p := Part{text: text, TsStartOffset: &tsStartOffset, TsEndOffset: &tsEndOffset}
	s.Parts = slices.Insert(s.Parts, 0, p)

	for _, p := range s.Parts {
		(*p.TsStartOffset) += tsStartOffset
		(*p.TsEndOffset) += tsEndOffset
	}
}

func (s *Statement) PugToTsLocation(start int, _end int) *Part {
	for _, part := range s.Parts {
		if part.PugStartOffset == nil || part.PugEndOffset == nil {
			continue
		}

		if *part.PugStartOffset <= start && *part.PugEndOffset > start {
			return &part
		}
	}

	return nil
}

func (s *Statement) TsToPugLocation(start int, _end int) *Part {
	for _, part := range s.Parts {
		if *part.TsStartOffset <= start && *part.TsEndOffset > start {
			return &part
		}
	}

	return nil
}

func (s *Statement) AddPart(part Part) {
	if part.node != nil && (part.PugStartOffset == nil || part.PugEndOffset == nil) {
		start := int(part.node.StartByte())
		end := int(part.node.EndByte())

		part.PugStartOffset = &start
		part.PugEndOffset = &end
	}

	s.Parts = append(s.Parts, part)
	s.sb.WriteString(part.text)
}

func (s *Statement) AddRealPart(text string, node *sitter.Node) {
	tsStartOffset := s.sb.Len()
	tsEndOffset := tsStartOffset + len(text)

	s.AddPart(Part{node: node, text: text, TsEndOffset: &tsEndOffset, TsStartOffset: &tsStartOffset})
}

func (s *Statement) AddScopePart(scope *Scope) {
	startOffset := 0
	if len(s.Parts) > 0 {
		startOffset = *s.Parts[len(s.Parts)-1].TsEndOffset
	}

	s.AddPart(Part{scope: scope, TsStartOffset: &startOffset})
}

func (s *Statement) AppendStatement(statement Statement) {
	s.AddStatement(&statement)
	s.AddVirtPart("\n")
}

func (s *Statement) CloseScopePart() {
	l := s.sb.Len()
	endOffset := *s.Parts[len(s.Parts)-1].TsStartOffset + l
	s.Parts[len(s.Parts)-1].TsEndOffset = &endOffset
}

func (s *Statement) ToString() string {
	sb := strings.Builder{}
	for _, p := range s.Parts {
		sb.WriteString(p.text)
	}

	return sb.String()
}

func StatementFromNodeContent(node *sitter.Node, content []byte) *Statement {
	nodeContent := node.Content(content)

	statement := Statement{}
	statement.AddRealPart(nodeContent, node)

	return &statement
}

func StatementFromString(text string) *Statement {
	statement := Statement{}
	statement.AddVirtPart(text)

	return &statement
}
