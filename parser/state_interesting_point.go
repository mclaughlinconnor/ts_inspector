package parser

import (
	"strconv"
	"ts_inspector/interfaces"
)

type InterestingPoint struct {
	Location interfaces.Location
	Text     string
	Kind     interfaces.TSymbolKind
}

func (i InterestingPoint) Id() string {
	return i.Location.Uri + strconv.FormatUint(uint64(i.Location.Range.Start.Line), 10) + strconv.FormatUint(uint64(i.Location.Range.Start.Character), 10) + strconv.FormatUint(uint64(i.Location.Range.End.Line), 10) + strconv.FormatUint(uint64(i.Location.Range.End.Character), 10)
}
