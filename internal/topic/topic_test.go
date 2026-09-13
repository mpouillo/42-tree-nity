package topic

import (
	"testing"

	"github.com/stretchr/testify/assert"

	trie "github.com/mpouillo/42-tree-nity/internal/structs/trie"
)

func TestNewTopic(t *testing.T) {
	got := NewTopic("test")
	want := &Topic{
		"test",
		*trie.NewTrie(),
		[]Message{},
	}

	assert.Equal(t, want, got)
}

func TestNewMessage(t *testing.T) {
	got := NewMessage("user.update","test")
	want := &Message{
		"user.update",
		"test",
	}

	assert.Equal(t, want, got)
}

func TestPush(t *testing.T) {
	topic := NewTopic("test")
	topic.Push("user.input:hello:world!")

	want := &Topic{
		"test",
		*trie.NewTrie(),
		[]Message{*NewMessage("user.input", "hello:world!")},
	}

	assert.Equal(t, want, topic)
}
