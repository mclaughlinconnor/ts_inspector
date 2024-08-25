package utils

// WARNING: not thread-safe

type Stack[T any] struct {
	data []T
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{make([]T, 0)}
}

func (s *Stack[T]) Push(v T) {
	s.data = append(s.data, v)
}

func (s *Stack[T]) Pop() *T {
	lenght := len(s.data)
	if lenght == 0 {
		return nil
	}

	top := &s.data[lenght-1]
	s.data = s.data[:lenght-1]

	return top
}

func (s *Stack[T]) IsEmpty() bool {
	return len(s.data) == 0
}

func (s *Stack[T]) Peek() *T {
	lenght := len(s.data)
	if lenght == 0 {
		return nil
	}

	top := &s.data[lenght-1]

	return top
}
