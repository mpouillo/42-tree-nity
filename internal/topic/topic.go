package topic

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/mpouillo/42-tree-nity/internal/consumer"
	"github.com/mpouillo/42-tree-nity/internal/message"
	"github.com/mpouillo/42-tree-nity/internal/structs/trie"
)

var (
	validName          = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,32}$`)
	ErrDuplicateClient = errors.New("client already exists")
	ErrTopicClosed     = errors.New("topic is closed")
)

type Topic struct {
	Name         string
	mu           sync.RWMutex
	messages     []message.TopicMessage
	nextOffset   uint32
	prefixTrie   *trie.Trie[*consumer.Consumer]
	consumers    map[string]*consumer.Consumer
	msgChan      chan message.TopicMessage
	subChan      chan *consumer.Consumer // consumers that just subscribed
	ackChan      chan *consumer.Consumer // consumers that acknowledged their last message
	stopped      chan struct{}           // closed when run() returns
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	OnDisconnect func(clientID string)
}

func NewTopic(parentCtx context.Context, name string) (*Topic, error) {
	if !validName.MatchString(name) {
		return nil, fmt.Errorf("invalid name %q (expected %s)", name, validName)
	}

	ctx, cancel := context.WithCancel(parentCtx)

	t := &Topic{
		Name:       name,
		messages:   make([]message.TopicMessage, 0),
		nextOffset: 0,
		prefixTrie: trie.NewTrie[*consumer.Consumer](),
		consumers:  make(map[string]*consumer.Consumer),
		msgChan:    make(chan message.TopicMessage, 1024),
		subChan:    make(chan *consumer.Consumer, 1024),
		ackChan:    make(chan *consumer.Consumer, 1024),
		stopped:    make(chan struct{}),
		ctx:        ctx,
		cancel:     cancel,
	}

	t.wg.Add(1)
	go t.run()

	log.Debugf("created topic %v", t.Name)
	return t, nil
}

func (t *Topic) run() {
	defer t.wg.Done()
	defer close(t.stopped)

	for {
		select {
		case msg := <-t.msgChan:
			t.processAndDispatch(msg)
		case c := <-t.subChan:
			if t.isSubscribed(c) {
				t.sendNext(c)
			}
		case c := <-t.ackChan:
			if !t.isSubscribed(c) {
				continue
			}
			c.InFlight = false
			t.sendNext(c)
		case <-t.ctx.Done():
			t.drainAndFlush()
			return
		}
	}
}

func (t *Topic) Close() error {
	t.cancel()
	t.wg.Wait()

	log.Debugf("closed topic %v", t.Name)
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
	t.mu.Lock()
	if t.ctx.Err() != nil {
		t.mu.Unlock()
		return ErrTopicClosed
	}
	if _, exists := t.consumers[c.ID]; exists {
		t.mu.Unlock()
		return ErrDuplicateClient
	}

	c.InFlight = false

	t.consumers[c.ID] = c
	if c.Prefix != "" {
		t.prefixTrie.Insert(c.Prefix, c)
	}
	t.mu.Unlock()

	select {
	case t.subChan <- c:
	case <-t.stopped:
		return ErrTopicClosed
	}

	log.Debugf("subscribed consumer %v to topic %v", c.ID, t.Name)
	return nil
}

func (t *Topic) Ack(c *consumer.Consumer) error {
	select {
	case t.ackChan <- c:
		return nil
	case <-t.stopped:
		return ErrTopicClosed
	}
}

func (t *Topic) HasConsumer(clientID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	_, exists := t.consumers[clientID]
	return exists
}

func (t *Topic) isSubscribed(c *consumer.Consumer) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.consumers[c.ID] == c
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
	log.Debugf("unsubscribed consumer %v from topic %v", c.ID, t.Name)
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

// a new message only wakes up the consumers it matches, the others stay idle
func (t *Topic) processAndDispatch(msg message.TopicMessage) {
	for _, c := range t.MatchConsumers(string(msg.Key)) {
		t.sendNext(c)
	}
}

// sendNext sends to the consumer the first message matching its prefix from
// its committed offset, unless it is still waiting for an ack. Must only be called from run().
func (t *Topic) sendNext(c *consumer.Consumer) {
	if c.InFlight {
		return
	}

	// offsets start at 0 and are sequential: the index of a message is its offset
	t.mu.RLock()
	var next *message.TopicMessage
	for i := int(c.Offset.Load()); next == nil && i < len(t.messages); i++ {
		if bytes.HasPrefix(t.messages[i].Key, []byte(c.Prefix)) {
			msg := t.messages[i]
			next = &msg
		}
	}
	t.mu.RUnlock()

	if next == nil {
		return
	}

	if err := c.Deliver(*next); err != nil {
		log.Errorf("failed to deliver message %v to consumer %v: %v", *next, c.ID, err)
		t.Unsubscribe(c.ID)
		if t.OnDisconnect != nil {
			t.OnDisconnect(c.ID)
		}
		return
	}
	c.InFlight = true
	log.Debugf("dispatched message %v to consumer %v", *next, c.ID)
}

func (t *Topic) drainAndFlush() {
	for len(t.msgChan) > 0 {
		msg := <-t.msgChan
		t.processAndDispatch(msg)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	for _, c := range t.consumers {
		if c.Prefix != "" {
			t.prefixTrie.Remove(c.Prefix, c, func(a, b *consumer.Consumer) bool { return a.ID == b.ID })
		}
		if t.OnDisconnect != nil {
			t.OnDisconnect(c.ID)
		}
		_ = c.CloseIPCChannel()
	}

	t.consumers = make(map[string]*consumer.Consumer)
}

func (t *Topic) GetMessages() []message.TopicMessage {
	return slices.Clone(t.messages)
}
