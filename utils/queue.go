package utils

// WARNING: not thread-safe

type Queue[T any] struct {
	data []T
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{make([]T, 0)}
}

func (s *Queue[T]) Push(v T) {
	s.data = append(s.data, v)
}

func (s *Queue[T]) Pop() *T {
	length := len(s.data)

	if length == 0 {
		return nil
	}

	top := &s.data[0]
	s.data = s.data[1:]

	return top
}

func (s *Queue[T]) IsEmpty() bool {
	return len(s.data) == 0
}

func (s *Queue[T]) Peek() *T {
	length := len(s.data)
	if length == 0 {
		return nil
	}

	top := &s.data[0]

	return top
}
