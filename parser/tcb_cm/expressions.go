package tcb_cm

type StatementParts struct {
	Parts []string
}

func (s *StatementParts) AddCodeBlock(builder func()) {
	s.Parts = append(s.Parts, "{\n")
	builder()
	s.Parts = append(s.Parts, "\n}")
}

func (s *StatementParts) AddPart(part string) {
	s.Parts = append(s.Parts, part)
}

func (s *StatementParts) AppendStatement(statement StatementParts) {
	s.Parts = append(s.Parts, statement.Parts...)
	s.Parts = append(s.Parts, "\n")
}

type Expression struct {
	Parts []StatementParts
}
