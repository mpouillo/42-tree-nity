package trie

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTrie(t *testing.T) {
	trie := NewTrie()

	if trie == nil {
		t.Fatal("Expected NewTrie() to return a non-nil Trie instance")
	}
	if trie.RootNode == nil {
		t.Fatal("Expected RootNode to be initialized, got nil")
	}
	if trie.RootNode.Values != nil {
		t.Errorf("Expected root node values to be nil/empty, got %v", trie.RootNode.Values)
	}
}

type client struct {
	id     string
	offset int64
}

func TestInsert(t *testing.T) {
	t.Run("Basic insert", func(t *testing.T) {
		key := "user"
		value := client{"client0", 0}

		trie := NewTrie()
		trie.Insert(key, value)

		current := trie.RootNode
		for _, char := range key {
			current = current.Children[char]
		}
		assert.Equal(t, []any{value}, current.Values)
	})

	t.Run("Root insert", func(t *testing.T) {
		key := ""
		value := client{"client0", 0}

		trie := NewTrie()
		trie.Insert(key, value)

		current := trie.RootNode
		assert.Equal(t, []any{value}, current.Values)
	})
}

func TestSearch(t *testing.T) {
	trie := NewTrie()
	trie.Insert("user", client{"client0", 0})
	trie.Insert("user.action", client{"client1", 3})
	trie.Insert("", client{"client2", 99})

	got, _ := trie.Search("user.action")
	want := []any{
		client{"client2", 99},
		client{"client0", 0},
		client{"client1", 3},
	}

	assert.ElementsMatch(t, want, got)
}