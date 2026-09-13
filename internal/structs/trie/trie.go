package trie

type Node struct {
	Value    any
	Children map[rune]*Node
}

func NewNode(value any) *Node {
	return &Node{
		Value:    value,
		Children: make(map[rune]*Node),
	}
}

type Trie struct {
	RootNode *Node
}

func NewTrie() *Trie {
	root := NewNode(nil)
	return &Trie{RootNode: root}
}

func (t *Trie) Insert(key string, value any) {
	current := t.RootNode
	for _, char := range key {
		if _, exists := current.Children[char]; !exists {
			current.Children[char] = NewNode(nil)
		}
		current = current.Children[char]
	}
	current.Value = value
}

func (t *Trie) Search(key string) ([]any, error) {
    current := t.RootNode
    var matches []any

    if current.Value != nil {
        matches = append(matches, current.Value)
    }

    for _, char := range key {
        next, exists := current.Children[char]
        if !exists {
            return matches, nil
        }
        current = next

        if current.Value != nil {
            matches = append(matches, current.Value)
        }
    }

    return matches, nil
}