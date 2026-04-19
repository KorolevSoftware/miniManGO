package main

type Stack[T any] struct {
	elements []T
}

func (s *Stack[T]) Push(value T) {
	s.elements = append(s.elements, value)
}

func (s *Stack[T]) Pop() (T, bool) {
	if len(s.elements) == 0 {
		var zero T
		return zero, false
	}
	index := len(s.elements) - 1
	element := s.elements[index]
	s.elements = s.elements[:index]
	return element, true
}

func (s *Stack[T]) Peek() (T, bool) {
	if len(s.elements) == 0 {
		var zero T
		return zero, false
	}
	return s.elements[len(s.elements)-1], true
}

func (s *Stack[T]) Size() int {
	return len(s.elements)
}
