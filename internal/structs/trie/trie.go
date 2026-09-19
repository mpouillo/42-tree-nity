package trie

import "slices"

type Node[T any] struct {
	Values   []T
	Children map[rune]*Node[T]
}

func NewNode[T any]() *Node[T] {
	return &Node[T]{
		Values:   make([]T, 0),
		Children: make(map[rune]*Node[T]),
	}
}

type Trie[T any] struct {
	RootNode *Node[T]
}

func NewTrie[T any]() *Trie[T] {
	return &Trie[T]{RootNode:NewNode[T]()}
}

func (t *Trie[T]) Insert(key string, value T) {
	current := t.RootNode

	for _, char := range key {
		if _, exists := current.Children[char]; !exists {
			current.Children[char] = NewNode[T]()
		}
		current = current.Children[char]
	}
	current.Values = append(current.Values, value)
}

func (t *Trie[T]) Remove(key string, target T, equals func(a, b T) bool) {
	current := t.RootNode

	for _, char := range key {
		next, exists := current.Children[char]
		if !exists {
			return
		}
		current = next
	}
	current.Values = slices.DeleteFunc(current.Values, func(val T) bool {
		return equals(val, target)
	})
}

func (t *Trie[T]) Search(key string) []T {
	current := t.RootNode
	var matches []T

	matches = append(matches, current.Values...)

	for _, char := range key {
		next, exists := current.Children[char]
		if !exists {
			return matches
		}
		current = next
		matches = append(matches, current.Values...)
	}

	return matches
}
