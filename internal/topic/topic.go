package topic

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/mpouillo/42-tree-nity/internal/consumer"
	message "github.com/mpouillo/42-tree-nity/internal/message"
	trie "github.com/mpouillo/42-tree-nity/internal/structs/trie"
)

var ErrDuplicateClient = errors.New("client already exists")
var ErrTopicClosed = errors.New("topic is closed")

type Topic struct {
	Name       string
	mu         sync.RWMutex
	messages   []message.TopicMessage
	nextOffset uint32
	prefixTrie *trie.Trie[*consumer.Consumer]
	consumers  map[string]*consumer.Consumer
	msgChan    chan message.TopicMessage
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

func NewTopic(parentCtx context.Context, name string) *Topic {
	ctx, cancel := context.WithCancel(parentCtx)

	t := &Topic{
		Name:       name,
		messages:   make([]message.TopicMessage, 0),
		nextOffset: 0,
		prefixTrie: trie.NewTrie[*consumer.Consumer](),
		consumers:  make(map[string]*consumer.Consumer),
		msgChan:    make(chan message.TopicMessage, 1024),
		ctx:        ctx,
		cancel:     cancel,
	}

	t.wg.Add(1)
	go t.run()
	return t
}

func (t *Topic) run() {
	defer t.wg.Done()

	for {
		select {
		case msg := <-t.msgChan:
			t.processAndDispatch(msg)
		case <-t.ctx.Done():
			t.drainAndFlush()
			return
		}
	}
}

func (t *Topic) Close() error {
	t.cancel()
	t.wg.Wait()
	return nil
}

func (t *Topic) Produce(key []byte, body []byte) (uint32, error) {
	if t.ctx.Err() != nil {
		return 0, ErrTopicClosed
	}

	t.mu.Lock()
	offset := t.nextOffset
	t.nextOffset++

	msg := message.TopicMessage{
		Key:    key,
		Body:   body,
		Offset: offset,
	}

	t.messages = append(t.messages, msg)
	t.mu.Unlock()

	select {
	case t.msgChan <- msg:
		return offset, nil
	case <-t.ctx.Done():
		return 0, ErrTopicClosed
	}
}

func (t *Topic) Subscribe(c *consumer.Consumer) error {
	if t.ctx.Err() != nil {
		return ErrTopicClosed
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.consumers[c.ID]; exists {
		return ErrDuplicateClient
	}

	// Catch up consumer
	currentOffset := c.Offset.Load()
	for _, msg := range t.messages {
		if msg.Offset >= currentOffset {
			if c.Prefix == "" || strings.HasPrefix(string(msg.Key), c.Prefix) {
				if err := c.Deliver(msg); err != nil {
					return err
				}
				c.Offset.Store(msg.Offset + 1)
			}
		}
	}

	t.consumers[c.ID] = c
	if c.Prefix != "" {
		t.prefixTrie.Insert(c.Prefix, c)
	}
	return nil
}

func (t *Topic) Unsubscribe(clientID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	c, exists := t.consumers[clientID]
	if !exists {
		return
	}

	if c.Prefix != "" {
		t.prefixTrie.Remove(c.Prefix, c, func(a, b *consumer.Consumer) bool { return a.ID == b.ID })
	}

	delete(t.consumers, clientID)
}

func (t *Topic) MatchConsumers(key string) []*consumer.Consumer {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var matched []*consumer.Consumer

	for _, c := range t.consumers {
		if c.Prefix == "" {
			matched = append(matched, c)
		}
	}

	prefixMatches := t.prefixTrie.Search(key)
	matched = append(matched, prefixMatches...)

	return matched
}

func (t *Topic) processAndDispatch(msg message.TopicMessage) {
	subscribers := t.MatchConsumers(string(msg.Key))

	for _, c := range subscribers {
		if msg.Offset >= c.Offset.Load() {
			if err := c.Deliver(msg); err != nil {
				t.Unsubscribe(c.ID)
				// TODO: notify server to remove client from hashmap
				continue
			}
			c.Offset.Store(msg.Offset + 1)
		}
	}
}

func (t *Topic) drainAndFlush() {
	for len(t.msgChan) > 0 {
		msg := <-t.msgChan
		t.processAndDispatch(msg)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	for id, c := range t.consumers {
		// remember to send error code 3 here when implemented
		_ = c.CloseIPCChannel()
		delete(t.consumers, id)
	}
}
