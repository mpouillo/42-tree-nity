package trie

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTrieInit[T any](t *testing.T) {
	t.Helper()
	trie := NewTrie[T]()

	require.NotNil(t, trie, "expected NewTrie() to return a non-nil Trie instance")
	require.NotNil(t, trie.RootNode, "expected RootNode to be initialized, got nil")
	assert.Empty(t,  trie.RootNode.Values)
}

func TestNewTrie(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		testTrieInit[string](t)
	})

	t.Run("int", func(t *testing.T) {
		testTrieInit[int](t)
	})

	t.Run("any / interface", func(t *testing.T) {
		testTrieInit[any](t)
	})

	t.Run("struct pointer", func(t *testing.T) {
		type custom struct{}
		testTrieInit[*custom](t)
	})
}

type client struct {
	id     string
	offset uint32
}

func TestInsert(t *testing.T) {
	t.Run("basic insert", func(t *testing.T) {
		key := "user"
		value := client{"client0", 0}

		trie := NewTrie[client]()
		trie.Insert(key, value)

		current := trie.RootNode
		for _, char := range key {
			next, exists := current.Children[char]
			require.Truef(t, exists, "expected child node for character %q to exist", char)
			current = next
		}
		assert.Equal(t, []client{value}, current.Values)
	})

	t.Run("root insert", func(t *testing.T) {
		key := ""
		value := client{"client0", 0}

		trie := NewTrie[client]()
		trie.Insert(key, value)

		current := trie.RootNode
		assert.Equal(t, []client{value}, current.Values)
	})
}

func TestRemove(t *testing.T) {
    t.Run("remove existing value", func(t *testing.T) {
        trie := NewTrie[client]()
        c0 := client{"client0", 0}
        c1 := client{"client1", 1}

        trie.Insert("user", c0)
        trie.Insert("user", c1)

        equals := func(a, b client) bool { return a.id == b.id }
        trie.Remove("user", c0, equals)

        got := trie.Search("user")
        assert.Equal(t, []client{c1}, got)
    })

    t.Run("remove non-existent key", func(t *testing.T) {
        trie := NewTrie[client]()
        c0 := client{"client0", 0}

        equals := func(a, b client) bool { return a.id == b.id }
        trie.Remove("missing", c0, equals)

        assert.Empty(t, trie.Search("missing"))
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