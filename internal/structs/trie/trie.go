package trie

type Node struct {
	Values    []any
	Children map[rune]*Node
}

func NewNode() *Node {
	return &Node{
		Values:    nil,
		Children: make(map[rune]*Node),
	}
}

type Trie struct {
	RootNode *Node
}

func NewTrie() *Trie {
	return &Trie{RootNode: NewNode()}
}

func (t *Trie) Insert(key string, value any) {
	current := t.RootNode
	for _, char := range key {
		if _, exists := current.Children[char]; !exists {
			current.Children[char] = NewNode()
		}
		current = current.Children[char]
	}
	current.Values = append(current.Values, value)
}

func (t *Trie) Search(key string) ([]any, error) {
    current := t.RootNode
    var matches []any

    if len(current.Values) > 0 {
        matches = append(matches, current.Values...)
    }

    for _, char := range key {
        next, exists := current.Children[char]
        if !exists {
            break
        }
        current = next

        if len(current.Values) > 0 {
            matches = append(matches, current.Values...)
        }
    }

    return matches, nil
}