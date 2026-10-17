package topic

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/mpouillo/42-tree-nity/internal/consumer"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
	"github.com/mpouillo/42-tree-nity/internal/message"
	"github.com/mpouillo/42-tree-nity/internal/structs/trie"
)

var (
	validName          = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,32}$`)
	ErrDuplicateClient = errors.New("client already exists")
	ErrTopicClosed     = errors.New("topic is closed")

	// ackTimeout is the delay after which a consumer that did not ack is disconnected.
	ackTimeout = 3 * time.Second

	// shutdownTimeout is how long Close waits for the last acks.
	shutdownTimeout = 5 * time.Second
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
	OnDisconnect func(c *consumer.Consumer, code uint8) // code is the exit code the client should use
}

// NewTopic returns an empty topic called name and starts its goroutine.
// The topic is closed when parentCtx is cancelled or Close is called.
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

// run is the goroutine of the topic: it dispatches messages, handles subscriptions
// and acks, and disconnects late consumers until the topic is closed.
func (t *Topic) run() {
	defer t.wg.Done()
	defer close(t.stopped)

	ticker := time.NewTicker(ackTimeout / 5)
	defer ticker.Stop()

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
		case <-ticker.C:
			t.disconnectUnacked()
		case <-t.ctx.Done():
			t.waitInFlightAcks()
			t.flush()
			return
		}
	}
}

// Close stops the topic after waiting up to shutdownTimeout for the in-flight acks,
// then tells every consumer to exit.
func (t *Topic) Close() error {
	t.cancel()
	t.wg.Wait()

	log.Debugf("closed topic %v", t.Name)
	return nil
}

// Produce stores a message with the next offset and returns that offset.
// It returns ErrTopicClosed if the topic is closed.
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

// Subscribe adds c to the topic, which starts sending it messages from its offset.
// It returns ErrDuplicateClient if a consumer with the same ID is subscribed.
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

// Ack tells the topic that c acknowledged its last message, so it can send the next one.
func (t *Topic) Ack(c *consumer.Consumer) error {
	select {
	case t.ackChan <- c:
		return nil
	case <-t.stopped:
		return ErrTopicClosed
	}
}

// HasConsumer reports whether a consumer named clientID is subscribed.
func (t *Topic) HasConsumer(clientID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	_, exists := t.consumers[clientID]
	return exists
}

// isSubscribed reports whether c itself, and not a newer consumer with its name, is subscribed.
func (t *Topic) isSubscribed(c *consumer.Consumer) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.consumers[c.ID] == c
}

// Unsubscribe removes the consumer named clientID, if any.
func (t *Topic) Unsubscribe(clientID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if c, exists := t.consumers[clientID]; exists {
		t.removeConsumer(c)
	}
}

// unsubscribe removes c unless a newer consumer reused its name, and reports
// whether it did.
func (t *Topic) unsubscribe(c *consumer.Consumer) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.consumers[c.ID] != c {
		return false
	}
	t.removeConsumer(c)
	return true
}

// removeConsumer removes c from the consumers and the prefix trie.
// The caller must hold t.mu.
func (t *Topic) removeConsumer(c *consumer.Consumer) {
	if c.Prefix != "" {
		t.prefixTrie.Remove(c.Prefix, c, func(a, b *consumer.Consumer) bool { return a.ID == b.ID })
	}

	delete(t.consumers, c.ID)
	log.Debugf("unsubscribed consumer %v from topic %v", c.ID, t.Name)
}

// MatchConsumers returns the consumers whose prefix matches key: the empty-prefix
// consumers, then those found in the prefix trie.
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

// processAndDispatch wakes up the consumers that match msg; the others stay idle.
func (t *Topic) processAndDispatch(msg message.TopicMessage) {
	for _, c := range t.MatchConsumers(string(msg.Key)) {
		t.sendNext(c)
	}
}

// sendNext sends c the first message at or after its offset that matches its
// prefix, unless one is already in flight. It must only be called from run.
func (t *Topic) sendNext(c *consumer.Consumer) {
	if c.InFlight {
		return
	}

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
		t.disconnect(c)
		return
	}
	c.InFlight = true
	c.SentAt = time.Now()
	log.Debugf("dispatched message %v to consumer %v", *next, c.ID)
}

// disconnectUnacked disconnects the consumers that did not ack within ackTimeout.
// It must only be called from run.
func (t *Topic) disconnectUnacked() {
	t.mu.RLock()
	var expired []*consumer.Consumer
	for _, c := range t.consumers {
		if c.InFlight && time.Since(c.SentAt) > ackTimeout {
			expired = append(expired, c)
		}
	}
	t.mu.RUnlock()

	for _, c := range expired {
		log.Warnf("consumer %v did not ack within %v", c.ID, ackTimeout)
		t.disconnect(c)
	}
}

// disconnect removes c and tells it to exit with an IPC error, unless a newer
// consumer reused its name.
func (t *Topic) disconnect(c *consumer.Consumer) {
	if !t.unsubscribe(c) {
		return
	}
	if t.OnDisconnect != nil {
		t.OnDisconnect(c, response.IpcError)
	}
	_ = c.CloseIPCChannel()
}

// waitInFlightAcks waits up to shutdownTimeout for the in-flight messages to be acked.
func (t *Topic) waitInFlightAcks() {
	deadline := time.After(shutdownTimeout)
	for t.hasInFlight() {
		select {
		case c := <-t.ackChan:
			c.InFlight = false
		case <-deadline:
			log.Warnf("topic %v: consumers did not ack within %v", t.Name, shutdownTimeout)
			return
		}
	}
}

// hasInFlight reports whether a consumer still has a message waiting for an ack.
func (t *Topic) hasInFlight() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, c := range t.consumers {
		if c.InFlight {
			return true
		}
	}
	return false
}

// flush removes every consumer and tells it to exit with code 0, as required
// for a graceful shutdown.
func (t *Topic) flush() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, c := range t.consumers {
		if c.Prefix != "" {
			t.prefixTrie.Remove(c.Prefix, c, func(a, b *consumer.Consumer) bool { return a.ID == b.ID })
		}
		if t.OnDisconnect != nil {
			t.OnDisconnect(c, response.NoError)
		}
		_ = c.CloseIPCChannel()
	}

	t.consumers = make(map[string]*consumer.Consumer)
}

// GetMessages returns a copy of the messages stored in the topic.
func (t *Topic) GetMessages() []message.TopicMessage {
	return slices.Clone(t.messages)
}
