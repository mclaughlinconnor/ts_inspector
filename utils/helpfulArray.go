package utils

type HelpfulArray[T any] struct {
	Elements []T
}

func (h *HelpfulArray[T]) Add(elem T) {
	h.Elements = append(h.Elements, elem)
}
