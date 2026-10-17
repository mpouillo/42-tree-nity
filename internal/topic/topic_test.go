package topic

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/mpouillo/42-tree-nity/internal/consumer"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/packet"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
	"github.com/mpouillo/42-tree-nity/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTopic checks that a new topic is named and empty.
func TestNewTopic(t *testing.T) {
	ctx := context.Background()
	got, err := NewTopic(ctx, "test")
	require.NoError(t, err)

	assert.Equal(t, "test", got.Name)
	assert.Equal(t, uint32(0), got.nextOffset)
	assert.Empty(t, got.messages)

	_ = got.Close()
}

// TestNewTopic_InvalidName checks that a topic name outside the allowed format is rejected.
func TestNewTopic_InvalidName(t *testing.T) {
	ctx := context.Background()
	_, err := NewTopic(ctx, "invalid topic name with spaces!")
	assert.Error(t, err)
}

// TestProduce checks that a produced message is stored as is at offset 0.
func TestProduce(t *testing.T) {
	tests := []struct {
		name      string
		key, body string
	}{
		{"basic message", "user.update", "test"},
		{"empty message", "", ""},
		{"no key", "", "test"},
		{"no body", "user.update", ""},
		{"two separators 1", "user.update", ":test"},
		{"two separators 2", "user.update", "test:test2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			topic, err := NewTopic(ctx, "test")
			require.NoError(t, err)

			offset, err := topic.Produce([]byte(tt.key), []byte(tt.body))
			require.NoError(t, err)
			assert.Equal(t, uint32(0), offset)

			topic.mu.RLock()

			require.Len(t, topic.messages, 1)
			assert.Equal(t, []byte(tt.key), topic.messages[0].Key)
			assert.Equal(t, []byte(tt.body), topic.messages[0].Body)
			assert.Equal(t, uint32(0), topic.messages[0].Offset)

			topic.mu.RUnlock()
			_ = topic.Close()
		})
	}
}

// TestProduce_SequentialOffsets checks that offsets increase by one per message.
func TestProduce_SequentialOffsets(t *testing.T) {
	ctx := context.Background()
	topic, err := NewTopic(ctx, "test")
	require.NoError(t, err)

	o1, err := topic.Produce([]byte("k1"), []byte("v1"))
	require.NoError(t, err)
	assert.Equal(t, uint32(0), o1)

	o2, err := topic.Produce([]byte("k2"), []byte("v2"))
	require.NoError(t, err)
	assert.Equal(t, uint32(1), o2)

	o3, err := topic.Produce([]byte("k3"), []byte("v3"))
	require.NoError(t, err)
	assert.Equal(t, uint32(2), o3)

	_ = topic.Close()
}

// TestProduce_AfterClose checks that a closed topic refuses new messages.
func TestProduce_AfterClose(t *testing.T) {
	ctx := context.Background()
	topic, err := NewTopic(ctx, "test")
	require.NoError(t, err)

	require.NoError(t, topic.Close())

	_, err = topic.Produce([]byte("user.update"), []byte("payload"))
	assert.ErrorIs(t, err, ErrTopicClosed)
}

// TestMatchConsumers checks that a key matches the empty-prefix consumers and
// the consumers whose prefix it starts with.
func TestMatchConsumers(t *testing.T) {
	ctx := context.Background()
	topic, err := NewTopic(ctx, "test")
	require.NoError(t, err)

	cAll := &consumer.Consumer{ID: "c1", Prefix: ""}
	cUser := &consumer.Consumer{ID: "c2", Prefix: "user."}
	cOrder := &consumer.Consumer{ID: "c3", Prefix: "order."}

	topic.consumers[cAll.ID] = cAll
	topic.consumers[cUser.ID] = cUser
	topic.prefixTrie.Insert(cUser.Prefix, cUser)
	topic.consumers[cOrder.ID] = cOrder
	topic.prefixTrie.Insert(cOrder.Prefix, cOrder)

	matched := topic.MatchConsumers("user.created")
	assert.Len(t, matched, 2)
	assert.ElementsMatch(t, []*consumer.Consumer{cAll, cUser}, matched)

	matchedOrder := topic.MatchConsumers("order.shipped")
	assert.Len(t, matchedOrder, 2)
	assert.ElementsMatch(t, []*consumer.Consumer{cAll, cOrder}, matchedOrder)

	matchedBilling := topic.MatchConsumers("billing.paid")
	assert.Len(t, matchedBilling, 1)
	assert.Equal(t, cAll, matchedBilling[0])

	_ = topic.Close()
}

// TestSubscribe_DuplicateClient checks that a subscribed client name cannot subscribe again.
func TestSubscribe_DuplicateClient(t *testing.T) {
	ctx := context.Background()
	topic, err := NewTopic(ctx, "test")
	require.NoError(t, err)

	c := &consumer.Consumer{ID: "client-1"}
	topic.consumers[c.ID] = c

	err = topic.Subscribe(c)
	assert.ErrorIs(t, err, ErrDuplicateClient)

	_ = topic.Close()
}

// TestUnsubscribe checks that an unsubscribed consumer no longer matches any key.
func TestUnsubscribe(t *testing.T) {
	ctx := context.Background()
	topic, err := NewTopic(ctx, "test")
	require.NoError(t, err)

	c := &consumer.Consumer{ID: "client-1", Prefix: "user."}
	topic.consumers[c.ID] = c
	topic.prefixTrie.Insert(c.Prefix, c)

	topic.Unsubscribe("client-1")

	assert.Empty(t, topic.consumers)
	assert.Empty(t, topic.MatchConsumers("user.created"))

	_ = topic.Close()
}

// TestSubscribeAndDispatch_Integration checks that a subscriber receives data
// over its FIFO; the message is never acked.
func TestSubscribeAndDispatch_Integration(t *testing.T) {
	setShutdownTimeout(t, 50*time.Millisecond)
	pipePath, reader := setupTestFIFO(t)

	ctx := context.Background()
	topic, err := NewTopic(ctx, "test")
	require.NoError(t, err)

	_, err = topic.Produce([]byte("user.login"), []byte("historical"))
	require.NoError(t, err)

	c, err := consumer.NewConsumer("client-1", "test", "user.", 0, pipePath)
	require.NoError(t, err)

	err = topic.Subscribe(c)
	require.NoError(t, err)

	_, err = topic.Produce([]byte("user.logout"), []byte("live"))
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	buf := make([]byte, 256)
	n, err := reader.Read(buf)
	require.NoError(t, err)
	assert.Greater(t, n, 0)

	_ = topic.Close()
	_ = reader.Close()
}

// TestPrefixFiltering_Integration checks which messages each prefix consumer
// receives, both when catching up on stored messages and live.
func TestPrefixFiltering_Integration(t *testing.T) {
	setShutdownTimeout(t, 100*time.Millisecond)

	topic, err := NewTopic(context.Background(), "test")
	require.NoError(t, err)
	defer func() { _ = topic.Close() }()

	history := []string{"user.create", "admin.login", "user.update"}
	live := []string{"user.create", "billing.paid", "user"}

	tests := []struct {
		prefix      string
		wantHistory []string
		wantLive    []string
	}{
		{"", history, live},
		{"user.", []string{"user.create", "user.update"}, []string{"user.create"}},
		{"user.create", []string{"user.create"}, []string{"user.create"}},
	}

	for _, key := range history {
		_, err := topic.Produce([]byte(key), []byte("body"))
		require.NoError(t, err)
	}

	consumers := make([]*consumer.Consumer, len(tests))
	readers := make([]*bufio.Reader, len(tests))
	for i, tt := range tests {
		pipePath, reader := setupTestFIFO(t)
		defer func() { _ = reader.Close() }()
		readers[i] = packet.NewReader(reader)

		consumers[i], err = consumer.NewConsumer(fmt.Sprintf("client-%d", i), "test", tt.prefix, 0, pipePath)
		require.NoError(t, err)
		require.NoError(t, topic.Subscribe(consumers[i]))
	}

	for i, tt := range tests {
		got := consume(t, topic, consumers[i], readers[i], len(tt.wantHistory))
		assert.Equal(t, tt.wantHistory, got, "history, prefix %q", tt.prefix)
	}

	for _, key := range live {
		_, err := topic.Produce([]byte(key), []byte("body"))
		require.NoError(t, err)
	}
	for i, tt := range tests {
		got := consume(t, topic, consumers[i], readers[i], len(tt.wantLive))
		assert.Equal(t, tt.wantLive, got, "live, prefix %q", tt.prefix)

		_, err := readDelivered(t, readers[i], 100*time.Millisecond)
		assert.ErrorIs(t, err, context.DeadlineExceeded, "prefix %q received an unexpected message", tt.prefix)
	}
}

// consume receives n messages for c, acks each one and returns their keys in
// delivery order.
func consume(t *testing.T, topic *Topic, c *consumer.Consumer, r *bufio.Reader, n int) []string {
	t.Helper()
	var keys []string
	for range n {
		msg, err := readDelivered(t, r, time.Second)
		require.NoError(t, err, "consumer %s", c.ID)
		keys = append(keys, string(msg.Key))

		c.Offset.Store(msg.Offset + 1)
		require.NoError(t, topic.Ack(c))
	}
	return keys
}

// readDelivered returns the next message written on a consumer FIFO, or an
// error if none arrives within timeout.
func readDelivered(t *testing.T, r *bufio.Reader, timeout time.Duration) (message.TopicMessage, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	p, err := packet.ReadPacket(ctx, r)
	if err != nil {
		return message.TopicMessage{}, err
	}
	var resp response.Response
	require.NoError(t, json.Unmarshal(p.Payload, &resp))
	var data response.SubscribeData
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	return data.Message, nil
}

// TestAckTimeout_DisconnectsConsumer checks that a consumer that never acks,
// e.g. killed with SIGKILL, is disconnected.
func TestAckTimeout_DisconnectsConsumer(t *testing.T) {
	setAckTimeout(t, 50*time.Millisecond)
	pipePath, reader := setupTestFIFO(t)
	defer func() { _ = reader.Close() }()

	topic, err := NewTopic(context.Background(), "test")
	require.NoError(t, err)
	defer func() { _ = topic.Close() }()

	_, err = topic.Produce([]byte("user.login"), []byte("never acked"))
	require.NoError(t, err)

	c, err := consumer.NewConsumer("client-1", "test", "", 0, pipePath)
	require.NoError(t, err)
	require.NoError(t, topic.Subscribe(c))

	assert.Eventually(t, func() bool { return !topic.HasConsumer("client-1") },
		time.Second, 10*time.Millisecond)
}

// TestAckTimeout_KeepsIdleConsumer checks that a consumer with nothing in
// flight is never disconnected.
func TestAckTimeout_KeepsIdleConsumer(t *testing.T) {
	setAckTimeout(t, 50*time.Millisecond)
	pipePath, reader := setupTestFIFO(t)
	defer func() { _ = reader.Close() }()

	topic, err := NewTopic(context.Background(), "test")
	require.NoError(t, err)
	defer func() { _ = topic.Close() }()

	c, err := consumer.NewConsumer("client-1", "test", "", 0, pipePath)
	require.NoError(t, err)
	require.NoError(t, topic.Subscribe(c))

	time.Sleep(200 * time.Millisecond)
	assert.True(t, topic.HasConsumer("client-1"))
}

// TestClose_WaitsForInFlightAck checks that Close waits for the in-flight ack,
// then tells the consumer to exit with 0.
func TestClose_WaitsForInFlightAck(t *testing.T) {
	setShutdownTimeout(t, 2*time.Second)
	pipePath, reader := setupTestFIFO(t)
	defer func() { _ = reader.Close() }()

	topic, err := NewTopic(context.Background(), "test")
	require.NoError(t, err)
	codes := make(chan uint8, 1)
	topic.OnDisconnect = func(c *consumer.Consumer, code uint8) { codes <- code }

	_, err = topic.Produce([]byte("user.login"), []byte("in flight"))
	require.NoError(t, err)

	c, err := consumer.NewConsumer("client-1", "test", "", 0, pipePath)
	require.NoError(t, err)
	require.NoError(t, topic.Subscribe(c))

	buf := make([]byte, 256)
	_, err = reader.Read(buf)
	require.NoError(t, err)

	closed := make(chan struct{})
	go func() {
		_ = topic.Close()
		close(closed)
	}()

	select {
	case <-closed:
		t.Fatal("topic closed before the in-flight message was acked")
	case <-time.After(100 * time.Millisecond):
	}

	require.NoError(t, topic.Ack(c))
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("topic did not close after the ack")
	}
	assert.Equal(t, response.NoError, <-codes)
}

// TestClose_ShutdownTimeout checks that a consumer that never acks does not
// block Close longer than shutdownTimeout.
func TestClose_ShutdownTimeout(t *testing.T) {
	setShutdownTimeout(t, 100*time.Millisecond)
	pipePath, reader := setupTestFIFO(t)
	defer func() { _ = reader.Close() }()

	topic, err := NewTopic(context.Background(), "test")
	require.NoError(t, err)

	_, err = topic.Produce([]byte("user.login"), []byte("never acked"))
	require.NoError(t, err)

	c, err := consumer.NewConsumer("client-1", "test", "", 0, pipePath)
	require.NoError(t, err)
	require.NoError(t, topic.Subscribe(c))

	buf := make([]byte, 256)
	_, err = reader.Read(buf)
	require.NoError(t, err)

	start := time.Now()
	require.NoError(t, topic.Close())
	assert.Less(t, time.Since(start), time.Second)
}

// setAckTimeout sets ackTimeout to d for the duration of the test. It must be
// called before NewTopic, whose goroutine reads it.
func setAckTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	previous := ackTimeout
	ackTimeout = d
	t.Cleanup(func() { ackTimeout = previous })
}

// setShutdownTimeout sets shutdownTimeout to d for the duration of the test.
func setShutdownTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	previous := shutdownTimeout
	shutdownTimeout = d
	t.Cleanup(func() { shutdownTimeout = previous })
}

// setupTestFIFO creates a FIFO and returns its path with a read end, opened
// read-write so that it never blocks and NewConsumer can open it for writing.
func setupTestFIFO(t *testing.T) (string, *os.File) {
	t.Helper()
	dir := t.TempDir()
	pipePath := filepath.Join(dir, "test.fifo")

	err := syscall.Mkfifo(pipePath, 0600)
	require.NoError(t, err)

	r, err := os.OpenFile(pipePath, os.O_RDWR, 0600)
	require.NoError(t, err)

	return pipePath, r
}
