package utils

type Range struct {
	Start Position `json:"start"`

	End Position `json:"end"`
}

func ZeroRange() Range {
	return Range{ZeroPosition(), ZeroPosition()}
}
