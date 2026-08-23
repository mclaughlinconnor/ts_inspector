package utils

type Range struct {
	Start Position `json:"start"`

	End Position `json:"end"`
}

func RangeFromStart(start Position) Range {
	return Range{start, start}
}

func ZeroRange() Range {
	return Range{ZeroPosition(), ZeroPosition()}
}
