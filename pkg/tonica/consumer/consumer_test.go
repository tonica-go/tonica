package consumer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tonica-go/tonica/pkg/tonica/storage"
	"github.com/tonica-go/tonica/pkg/tonica/storage/pubsub"
)

// Mock pubsub client for testing
type mockPubSubClient struct {
	messages       []*pubsub.Message
	messageIndex   int
	subscribeErr   error
	subscribeCalls int
}

func (m *mockPubSubClient) Subscribe(ctx context.Context, topic string) (*pubsub.Message, error) {
	m.subscribeCalls++

	if m.subscribeErr != nil {
		return nil, m.subscribeErr
	}

	if m.messageIndex >= len(m.messages) {
		// Wait for context cancellation
		<-ctx.Done()
		return nil, ctx.Err()
	}

	msg := m.messages[m.messageIndex]
	m.messageIndex++
	return msg, nil
}

func (m *mockPubSubClient) Publish(ctx context.Context, topic string, message []byte) error {
	return nil
}

func (m *mockPubSubClient) Health() storage.Health {
	return storage.Health{Status: storage.StatusUp}
}

func (m *mockPubSubClient) CreateTopic(ctx context.Context, name string) error {
	return nil
}

func (m *mockPubSubClient) DeleteTopic(ctx context.Context, name string) error {
	return nil
}

func (m *mockPubSubClient) Query(ctx context.Context, query string, args ...any) ([]byte, error) {
	return nil, nil
}

func (m *mockPubSubClient) Close() error {
	return nil
}

func TestConsumer_Start(t *testing.T) {
	t.Run("should process messages from subscribe", func(t *testing.T) {
		messages := []*pubsub.Message{
			{Value: []byte("message1")},
			{Value: []byte("message2")},
			{Value: []byte("message3")},
		}

		mockClient := &mockPubSubClient{
			messages: messages,
		}

		processedCount := 0
		handler := func(ctx context.Context, msg *pubsub.Message) error {
			processedCount++
			return nil
		}

		consumer := NewConsumer(
			WithName("test-consumer"),
			WithClient(mockClient),
			WithTopic("test-topic"),
			WithHandler(handler),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Start consumer in goroutine
		done := make(chan error)
		go func() {
			done <- consumer.Start(ctx)
		}()

		// Wait for completion or timeout
		select {
		case err := <-done:
			assert.Equal(t, context.DeadlineExceeded, err)
			assert.Equal(t, len(messages), processedCount, "should process all messages")
		case <-time.After(200 * time.Millisecond):
			t.Fatal("test timeout")
		}
	})

	t.Run("should handle context cancellation", func(t *testing.T) {
		mockClient := &mockPubSubClient{
			messages: []*pubsub.Message{
				{Value: []byte("message1")},
			},
		}

		handler := func(ctx context.Context, msg *pubsub.Message) error {
			return nil
		}

		consumer := NewConsumer(
			WithName("test-consumer"),
			WithClient(mockClient),
			WithTopic("test-topic"),
			WithHandler(handler),
		)

		ctx, cancel := context.WithCancel(context.Background())

		// Start consumer
		done := make(chan error)
		go func() {
			done <- consumer.Start(ctx)
		}()

		// Cancel after short delay
		time.Sleep(10 * time.Millisecond)
		cancel()

		// Should return context.Canceled
		select {
		case err := <-done:
			assert.Equal(t, context.Canceled, err)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("consumer did not stop on context cancellation")
		}
	})

	t.Run("should continue on subscribe error", func(t *testing.T) {
		mockClient := &mockPubSubClient{
			subscribeErr: errors.New("subscribe error"),
		}

		handler := func(ctx context.Context, msg *pubsub.Message) error {
			return nil
		}

		consumer := NewConsumer(
			WithName("test-consumer"),
			WithClient(mockClient),
			WithTopic("test-topic"),
			WithHandler(handler),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// Start consumer
		done := make(chan error)
		go func() {
			done <- consumer.Start(ctx)
		}()

		// Wait for timeout
		select {
		case err := <-done:
			assert.Equal(t, context.DeadlineExceeded, err)
			// Should have attempted multiple subscribes
			assert.Greater(t, mockClient.subscribeCalls, 1)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("test timeout")
		}
	})

	t.Run("should continue on handler error", func(t *testing.T) {
		messages := []*pubsub.Message{
			{Value: []byte("message1")},
			{Value: []byte("message2")},
		}

		mockClient := &mockPubSubClient{
			messages: messages,
		}

		processedCount := 0
		handler := func(ctx context.Context, msg *pubsub.Message) error {
			processedCount++
			return errors.New("handler error")
		}

		consumer := NewConsumer(
			WithName("test-consumer"),
			WithClient(mockClient),
			WithTopic("test-topic"),
			WithHandler(handler),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// Start consumer
		done := make(chan error)
		go func() {
			done <- consumer.Start(ctx)
		}()

		// Wait for completion
		select {
		case <-done:
			// Should process all messages despite errors
			assert.Equal(t, len(messages), processedCount)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("test timeout")
		}
	})
}

func TestNewConsumer(t *testing.T) {
	t.Run("should create consumer with options", func(t *testing.T) {
		mockClient := &mockPubSubClient{}
		handler := func(ctx context.Context, msg *pubsub.Message) error {
			return nil
		}

		consumer := NewConsumer(
			WithName("test-consumer"),
			WithClient(mockClient),
			WithTopic("test-topic"),
			WithConsumerGroup("test-group"),
			WithHandler(handler),
		)

		assert.NotNil(t, consumer)
		assert.Equal(t, "test-consumer", consumer.GetName())
		assert.Equal(t, "test-topic", consumer.GetTopic())
		assert.Equal(t, "test-group", consumer.GetConsumerGroup())
		assert.Same(t, mockClient, consumer.GetClient())
		assert.NotNil(t, consumer.GetHandler())
	})
}

func TestConsumer_Getters(t *testing.T) {
	mockClient := &mockPubSubClient{}
	handler := func(ctx context.Context, msg *pubsub.Message) error {
		return nil
	}

	consumer := &Consumer{
		name:          "test-name",
		topic:         "test-topic",
		consumerGroup: "test-group",
		client:        mockClient,
		handler:       handler,
	}

	t.Run("GetName", func(t *testing.T) {
		assert.Equal(t, "test-name", consumer.GetName())
	})

	t.Run("GetTopic", func(t *testing.T) {
		assert.Equal(t, "test-topic", consumer.GetTopic())
	})

	t.Run("GetConsumerGroup", func(t *testing.T) {
		assert.Equal(t, "test-group", consumer.GetConsumerGroup())
	})

	t.Run("GetClient", func(t *testing.T) {
		assert.Same(t, mockClient, consumer.GetClient())
	})

	t.Run("GetHandler", func(t *testing.T) {
		require.NotNil(t, consumer.GetHandler())

		// Test handler execution
		err := consumer.GetHandler()(context.Background(), &pubsub.Message{})
		assert.NoError(t, err)
	})
}
