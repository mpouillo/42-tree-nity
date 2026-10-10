package topic

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/mpouillo/42-tree-nity/internal/consumer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTopic(t *testing.T) {
	ctx := context.Background()
	got := NewTopic(ctx, "test")

	assert.Equal(t, "test", got.Name)
	assert.Equal(t, uint32(0), got.nextOffset)
	assert.Empty(t, got.messages)

	_ = got.Close()
}

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
			topic := NewTopic(ctx, "test")

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

func TestProduce_SequentialOffsets(t *testing.T) {
	ctx := context.Background()
	topic := NewTopic(ctx, "test")

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

func TestProduce_AfterClose(t *testing.T) {
	ctx := context.Background()
	topic := NewTopic(ctx, "test")

	require.NoError(t, topic.Close())

	_, err := topic.Produce([]byte("user.update"), []byte("payload"))
	assert.ErrorIs(t, err, ErrTopicClosed)
}

func TestMatchConsumers(t *testing.T) {
	ctx := context.Background()
	topic := NewTopic(ctx, "test")

	cAll := &consumer.Consumer{ID: "c1", Prefix: ""}
	cUser := &consumer.Consumer{ID: "c2", Prefix: "user."}
	cOrder := &consumer.Consumer{ID: "c3", Prefix: "order."}

	// Manually inject mock consumers into state for key-matching tests
	topic.consumers[cAll.ID] = cAll
	topic.consumers[cUser.ID] = cUser
	topic.prefixTrie.Insert(cUser.Prefix, cUser)
	topic.consumers[cOrder.ID] = cOrder
	topic.prefixTrie.Insert(cOrder.Prefix, cOrder)

	// "user.created" should match wildcard cAll and prefix cUser
	matched := topic.MatchConsumers("user.created")
	assert.Len(t, matched, 2)
	assert.ElementsMatch(t, []*consumer.Consumer{cAll, cUser}, matched)

	// "order.shipped" should match wildcard cAll and prefix cOrder
	matchedOrder := topic.MatchConsumers("order.shipped")
	assert.Len(t, matchedOrder, 2)
	assert.ElementsMatch(t, []*consumer.Consumer{cAll, cOrder}, matchedOrder)

	// "billing.paid" should match only wildcard cAll
	matchedBilling := topic.MatchConsumers("billing.paid")
	assert.Len(t, matchedBilling, 1)
	assert.Equal(t, cAll, matchedBilling[0])

	_ = topic.Close()
}

func TestSubscribe_DuplicateClient(t *testing.T) {
	ctx := context.Background()
	topic := NewTopic(ctx, "test")

	c := &consumer.Consumer{ID: "client-1"}
	topic.consumers[c.ID] = c

	err := topic.Subscribe(c)
	assert.ErrorIs(t, err, ErrDuplicateClient)

	_ = topic.Close()
}

func TestUnsubscribe(t *testing.T) {
	ctx := context.Background()
	topic := NewTopic(ctx, "test")

	c := &consumer.Consumer{ID: "client-1", Prefix: "user."}
	topic.consumers[c.ID] = c
	topic.prefixTrie.Insert(c.Prefix, c)

	topic.Unsubscribe("client-1")

	assert.Empty(t, topic.consumers)
	assert.Empty(t, topic.MatchConsumers("user.created"))

	_ = topic.Close()
}

// Integration test verifying Subscribe catch-up & live dispatch over an IPC FIFO
func TestSubscribeAndDispatch_Integration(t *testing.T) {
	pipePath, reader := setupTestFIFO(t)

	ctx := context.Background()
	topic := NewTopic(ctx, "test")

	// 1. Produce historical message before subscription
	_, err := topic.Produce([]byte("user.login"), []byte("historical"))
	require.NoError(t, err)

	// 2. Subscribe consumer
	c, err := consumer.NewConsumer("client-1", "test", "user.", 0, pipePath)
	require.NoError(t, err)

	err = topic.Subscribe(c)
	require.NoError(t, err)

	// 3. Produce live message after subscription
	_, err = topic.Produce([]byte("user.logout"), []byte("live"))
	require.NoError(t, err)

	// Give background dispatch loop a brief moment to write to the pipe
	time.Sleep(50 * time.Millisecond)

	buf := make([]byte, 256)
	n, err := reader.Read(buf)
	require.NoError(t, err)
	assert.Greater(t, n, 0)

	_ = topic.Close()
	_ = reader.Close()
}

// Helper to create a non-blocking FIFO reader for IPC delivery testing
func setupTestFIFO(t *testing.T) (string, *os.File) {
	t.Helper()
	dir := t.TempDir()
	pipePath := filepath.Join(dir, "test.fifo")

	err := syscall.Mkfifo(pipePath, 0600)
	require.NoError(t, err)

	// Open read-end non-blocking so NewConsumer write-open won't hang
	r, err := os.OpenFile(pipePath, os.O_RDWR, 0600)
	require.NoError(t, err)

	return pipePath, r
}
