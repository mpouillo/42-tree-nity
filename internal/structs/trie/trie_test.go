package trie

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTrieInit[T any](t *testing.T) {
	t.Helper()
	trie := NewTrie[T]()

	require.NotNil(t, trie, "expected NewTrie() to return a non-nil Trie instance")
	require.NotNil(t, trie.RootNode, "expected RootNode to be initialized, got nil")
	assert.Empty(t, trie.RootNode.Values)
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

	t.Run("mixed and overlapping prefixes", func(t *testing.T) {
		trie := NewTrie[string]()
		trie.Insert("use", "consumer-use")
		trie.Insert("user", "consumer-user")
		trie.Insert("username", "consumer-username")
		trie.Insert("user.login", "consumer-login")
		trie.Insert("useless", "consumer-useless")

		got := trie.Search("user.login")

		assert.Contains(t, got, "consumer-use")
		assert.Contains(t, got, "consumer-user")
		assert.Contains(t, got, "consumer-login")
		assert.NotContains(t, got, "consumer-username")
		assert.NotContains(t, got, "consumer-useless")
	})
}

func TestEmptyTrieEdgeCases(t *testing.T) {
	trie := NewTrie[int]()

	results := trie.Search("any.key")
	assert.Empty(t, results, "searching an empty trie should return no results")

	equals := func(a, b int) bool { return a == b }
	assert.NotPanics(t, func() {
		trie.Remove("any.key", 42, equals)
	}, "removing from an empty trie should not panic")
}

func TestLargeScaleTrie(t *testing.T) {
	trie := NewTrie[int]()

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("prefix.key.%d", i)
		trie.Insert(key, i)
	}

	results := trie.Search("prefix.key.500.subaction")
	assert.NotEmpty(t, results, "large scale search should return matching prefixes")
}

func TestEmptyPrefixWildcard(t *testing.T) {
	trie := NewTrie[string]()
	trie.Insert("", "wildcard-consumer")
	trie.Insert("user", "user-consumer")

	gotUnrelated := trie.Search("admin.settings.update")
	assert.Contains(t, gotUnrelated, "wildcard-consumer")
	assert.NotContains(t, gotUnrelated, "user-consumer")

	gotUser := trie.Search("user.login")
	assert.Contains(t, gotUser, "wildcard-consumer")
	assert.Contains(t, gotUser, "user-consumer")
}

func TestMultipleConsumersSamePrefix(t *testing.T) {
	trie := NewTrie[string]()
	trie.Insert("user.login", "client-alpha")
	trie.Insert("user.login", "client-beta")

	got := trie.Search("user.login")

	assert.Len(t, got, 2)
	assert.Contains(t, got, "client-alpha")
	assert.Contains(t, got, "client-beta")
}