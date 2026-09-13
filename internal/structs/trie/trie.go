package trie

import (
	"fmt"
	"regexp"
	"sort"
)

const pattern string = "^[a-zA-Z0-9_.-]{1,32}$"

type Node struct {
	Char     rune
	Children map[rune]*Node
	IsEnd    bool
}

func NewNode(r rune) *Node {
	return &Node{
		Char:     r,
		Children: make(map[rune]*Node),
	}
}

type Trie struct {
	RootNode *Node
}

func NewTrie() *Trie {
	root := NewNode(-1)
	return &Trie{RootNode: root}
}

func (t *Trie) Insert(word string) error {
	if len(word) < 1 || len(word) > 32 {
		return fmt.Errorf("word length must be between 1 and 32 characters")
	}

	current := t.RootNode
	for _, r := range word {
		if !isValidRune(r) {
			return fmt.Errorf("invalid character: %q", r)
		}

		if _, exists := current.Children[r]; !exists {
			current.Children[r] = NewNode(r)
		}
		current = current.Children[r]
	}
	current.IsEnd = true
	return nil
}

func isValidRune(r rune) bool {
	match, _ := regexp.MatchString(pattern, string(r))
	return match
}

func (t *Trie) SearchWord(word string) (bool, error) {
	if len(word) < 1 || len(word) > 32 {
		return false, fmt.Errorf("word length must be between 1 and 32 characters")
	}

	current := t.RootNode
	for _, r := range word {
		if !isValidRune(r) {
			return false, fmt.Errorf("invalid character: %q", r)
		}

		next, exists := current.Children[r]
		if !exists {
			return false, nil
		}
		current = next
	}
	return current.IsEnd, nil
}

func (t *Trie) SearchPrefix(word string) (bool, error) {
	if len(word) < 1 || len(word) > 32 {
		return false, fmt.Errorf("word length must be between 1 and 32 characters")
	}

	current := t.RootNode
	for _, r := range word {
		if !isValidRune(r) {
			return false, fmt.Errorf("invalid character: %q", r)
		}

		next, exists := current.Children[r]
		if !exists {
			return false, nil
		}
		current = next
	}
	return true, nil
}

type queueItem struct {
	node   *Node
	prefix string
}

func (t *Trie) FindMatches(prefix string) ([]string, error) {
	if len(prefix) < 1 || len(prefix) > 32 {
		return []string{}, fmt.Errorf("word length must be between 1 and 32 characters")
	}

	for _, r := range prefix {
		if !isValidRune(r) {
			return nil, fmt.Errorf("invalid character: %q", r)
		}
	}

	current := t.RootNode
	for _, r := range prefix {
		next, exists := current.Children[r]
		if !exists {
			return nil, nil
		}
		current = next
	}

	var results []string
	queue := []queueItem{{node: current, prefix: prefix}}

	for len(queue) > 0 {
		currItem := queue[0]
		queue = queue [1:]

		if currItem.node.IsEnd {
			results = append(results, currItem.prefix)
		}

		var keys []rune
		for k := range currItem.node.Children {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

		for _, r := range keys {
			childNode := currItem.node.Children[r]
			queue = append(queue, queueItem{
				node:	childNode,
				prefix: currItem.prefix + string(r),
			})
		}
	}

	return results, nil
}
