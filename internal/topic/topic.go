package topic

import (
	"strings"

	trie "github.com/mpouillo/42-tree-nity/internal/structs/trie"
)

type Message struct {
	key  string
	body string
}

func NewMessage(key, body string) *Message {
	return &Message{key, body}
}

type Topic struct {
	Name        string
	subscribers trie.Trie[string]
	messages    []Message
}

func NewTopic(name string) *Topic {
	return &Topic{
		Name: name,
		subscribers: *trie.NewTrie[string](),
		messages: []Message{},
	}
}

func (t *Topic) Push(message string) {
	key, body := splitKeyBody(message)
	t.messages = append(t.messages, *NewMessage(key, body))
}

func splitKeyBody(input string) (string, string) {
	parts := strings.SplitN(input, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}