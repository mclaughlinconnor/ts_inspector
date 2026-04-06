package tcb_cm

import (
	"slices"
	"strings"
)

type Expression struct {
	Parts []StatementParts
}

type Part struct {
	StartOffset *int
	EndOffset   *int
	Text        string
}

type StatementParts struct {
	Parts []Part
}

func (s *StatementParts) AddCodeBlock(builder func()) {
	s.Parts = append(s.Parts, Part{Text: "{\n"})
	builder()
	s.Parts = append(s.Parts, Part{Text: "\n}"})
}

func (s *StatementParts) AddVirtPart(text string) {
	s.AddRealPart(text, nil, nil)
}

func (s *StatementParts) AddStatementParts(statementParts *StatementParts) {
	for _, p := range statementParts.Parts {
		s.AddPart(p)
	}
}

func (s *StatementParts) PrependPart(part Part) {
	s.Parts = slices.Insert(s.Parts, 0, part)
}

func (s *StatementParts) AddPart(part Part) {
	s.Parts = append(s.Parts, part)
}

func (s *StatementParts) AddRealPart(text string, startOffset *int, endOffset *int) {
	s.AddPart(Part{StartOffset: startOffset, EndOffset: endOffset, Text: text})
}

func (s *StatementParts) AppendStatement(statement StatementParts) {
	s.Parts = append(s.Parts, statement.Parts...)
	s.Parts = append(s.Parts, Part{Text: "\n"})
}

func (s *StatementParts) ToString() string {
	sb := strings.Builder{}
	for _, p := range s.Parts {
		sb.WriteString(p.Text)
	}

	return sb.String()
}
