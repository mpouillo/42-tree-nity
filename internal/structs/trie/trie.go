package trie

type Node[T any] struct {
    Value    T
    Children map[rune]*Node[T]
}

func NewNode[T any](value T) *Node[T] {
    return &Node[T]{
        Value:    value,
        Children: make(map[rune]*Node[T]),
    }
}

type Trie[T any] struct {
    RootNode *Node[T]
}

func NewTrie[T any]() *Trie[T] {
    var zero T
    root := NewNode(zero)
    return &Trie[T]{RootNode: root}
}

func (t *Trie[T]) Insert(key string, value T) {
    current := t.RootNode
    var zero T

    for _, char := range key {
        if _, exists := current.Children[char]; !exists {
            current.Children[char] = NewNode(zero)
        }
        current = current.Children[char]
    }
    current.Value = value
}

func (t *Trie[T]) Search(key string) []T {
    current := t.RootNode
    var matches []T
    var zero T

    if any(current.Value) != any(zero) {
        matches = append(matches, current.Value)
    }

    for _, char := range key {
        next, exists := current.Children[char]
        if !exists {
            return matches
        }
        current = next

        if any(current.Value) != any(zero) {
            matches = append(matches, current.Value)
        }
    }

    return matches
}