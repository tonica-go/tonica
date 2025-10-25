package consumer

import (
	"context"
	"log/slog"

	"github.com/tonica-go/tonica/pkg/tonica/storage/pubsub"
)

type Handler func(ctx context.Context, msg *pubsub.Message) error

type Consumer struct {
	client        pubsub.Client
	name          string
	consumerGroup string
	topic         string
	handler       func(ctx context.Context, msg *pubsub.Message) error
}

func (c *Consumer) GetName() string {
	return c.name
}

func (c *Consumer) GetClient() pubsub.Client {
	return c.client
}

func (c *Consumer) GetConsumerGroup() string {
	return c.consumerGroup
}

func (c *Consumer) GetTopic() string {
	return c.topic
}

func (c *Consumer) GetHandler() func(ctx context.Context, msg *pubsub.Message) error {
	return c.handler
}

type Option func(*Consumer)

func WithName(name string) Option {
	return func(a *Consumer) {
		a.name = name
	}
}

func WithClient(c pubsub.Client) Option {
	return func(a *Consumer) {
		a.client = c
	}
}

func WithConsumerGroup(c string) Option {
	return func(a *Consumer) {
		a.consumerGroup = c
	}
}

func WithTopic(c string) Option {
	return func(a *Consumer) {
		a.topic = c
	}
}

func WithHandler(c func(ctx context.Context, msg *pubsub.Message) error) Option {
	return func(a *Consumer) {
		a.handler = c
	}
}

func NewConsumer(options ...Option) *Consumer {
	app := &Consumer{}
	for _, option := range options {
		option(app)
	}

	return app
}

func (c *Consumer) Start(ctx context.Context) error {
	for msg, err := c.client.Subscribe(ctx, c.topic); err == nil; {
		err := c.handler(ctx, msg)
		if err != nil {
			slog.Error("handling consumer message failed", "err", err.Error())
		}
	}

	return nil
}
