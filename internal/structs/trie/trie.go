package trie

import "slices"

type Node[T any] struct {
	Values   []T
	Children map[rune]*Node[T]
}

// NewNode returns a node with no values and no children.
func NewNode[T any]() *Node[T] {
	return &Node[T]{
		Values:   make([]T, 0),
		Children: make(map[rune]*Node[T]),
	}
}

type Trie[T any] struct {
	RootNode *Node[T]
}

// NewTrie returns an empty trie.
func NewTrie[T any]() *Trie[T] {
	return &Trie[T]{RootNode:NewNode[T]()}
}

// Insert adds value under key; several values can share the same key.
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

// Remove deletes the values stored under key that are equal to target.
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

// Search returns the values of every key that is a prefix of key, shortest first.
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
