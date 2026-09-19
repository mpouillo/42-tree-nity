package trie

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTrieInit[T comparable](t *testing.T, zero T) {
	trie := NewTrie[T]()

	require.NotNil(t, trie, "Expected NewTrie() to return a non-nil Trie instance")
	require.NotNil(t, trie.RootNode, "Expected RootNode to be initialized, got nil")
	assert.Equal(t, zero, trie.RootNode.Value)
}

func TestNewTrie(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		testTrieInit[string](t, "")
	})

	t.Run("int", func(t *testing.T) {
		testTrieInit[int](t, 0)
	})

	t.Run("any / interface", func(t *testing.T) {
		testTrieInit[any](t, nil)
	})

	t.Run("struct pointer", func(t *testing.T) {
		type custom struct{}
		testTrieInit[*custom](t, nil)
	})
}

type client struct {
	id     string
	offset int64
}

func TestInsert(t *testing.T) {
	t.Run("Basic insert", func(t *testing.T) {
		key := "user"
		value := client{"client0", 0}

		trie := NewTrie[client]()
		trie.Insert(key, value)

		current := trie.RootNode
		for _, char := range key {
			next, exists := current.Children[char]
			require.Truef(t, exists, "Expected child node for character %q to exist", char)
			current = next
		}
		assert.Equal(t, value, current.Value)
	})

	t.Run("Root insert", func(t *testing.T) {
		key := ""
		value := client{"client0", 0}

		trie := NewTrie[client]()
		trie.Insert(key, value)

		current := trie.RootNode
		assert.Equal(t, value, current.Value)
	})
}

func TestSearch(t *testing.T) {
	t.Run("prefix match", func(t *testing.T) {
		trie := NewTrie[client]()
		trie.Insert("user", client{"client0", 0})
		trie.Insert("user.action", client{"client1", 3})
		trie.Insert("", client{"client2", 99})

		got := trie.Search("user.action")

		want := []client{
			{"client2", 99},
			{"client0", 0},
			{"client1", 3},
		}

		assert.Equal(t, want, got)
	})

	t.Run("no prefix match", func(t *testing.T) {
		trie := NewTrie[client]()
		trie.Insert("user.action", client{"client1", 3})

		got := trie.Search("missing")
		assert.Empty(t, got)
	})
}