package tcb_cm

type HelpfulArray[T any] struct {
	Elements []T
}

func (h *HelpfulArray[T]) add(elem T) {
	h.Elements = append(h.Elements, elem)
}
