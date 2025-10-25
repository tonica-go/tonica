package pubsub

import (
	"context"

	"github.com/tonica-go/tonica/pkg/tonica/storage"
)

type Publisher interface {
	Publish(ctx context.Context, topic string, message []byte) error
}

type Subscriber interface {
	Subscribe(ctx context.Context, topic string) (*Message, error)
}

type Client interface {
	Publisher
	Subscriber
	Health() storage.Health

	CreateTopic(context context.Context, name string) error
	DeleteTopic(context context.Context, name string) error
	Query(ctx context.Context, query string, args ...any) ([]byte, error)

	Close() error
}

type Committer interface {
	Commit()
}
