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
		*trie.NewTrie[string](),
		[]Message{},
	}

	assert.Equal(t, want, got)
}

func TestNewMessage(t *testing.T) {
	tests := []struct {
		name      string
		key, body string
	}{
		{
			"empty message",
			"", "",
		},
		{
			"no key",
			"", "test",
		},
		{
			"no body",
			"user.update", "",
		},
				{
			"regular message",
			"user.update", "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewMessage(tt.key, tt.body)
			want := &Message{tt.key, tt.body}

			assert.Equal(t, want, got)
		})
	}
}

func TestPush(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		key, body string
	}{
		{
			"basic message",
			"user.update:test",
			"user.update", "test",
		},
		{
			"empty message",
			"",
			"", "",
		},
		{
			"only separator",
			":",
			"", "",
		},
		{
			"no key",
			":test",
			"", "test",
		},
		{
			"no body",
			"user.update:",
			"user.update", "",
		},
		{
			"two separators 1",
			"user.update::test",
			"user.update", ":test",
		},
		{
			"two separators 2",
			"user.update:test:test2",
			"user.update", "test:test2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic := NewTopic("test")
			topic.Push(tt.message)

			want := &Topic{
				"test",
				*trie.NewTrie[string](),
				[]Message{*NewMessage(tt.key, tt.body)},
			}

			assert.Equal(t, want, topic)
		})
	}
}