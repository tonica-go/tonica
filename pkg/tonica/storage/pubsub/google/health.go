package google

import (
	"context"
	"errors"
	"time"

	"github.com/tonica-go/tonica/pkg/tonica/storage"
	"google.golang.org/api/iterator"
)

func (g *googleClient) Health() (health storage.Health) {
	health.Details = make(map[string]any)

	var writerStatus, readerStatus string

	health.Status = storage.StatusUp
	health.Details["projectID"] = g.Config.ProjectID
	health.Details["backend"] = "GOOGLE"

	writerStatus, health.Details["writers"] = g.getWriterDetails()
	readerStatus, health.Details["readers"] = g.getReaderDetails()

	if readerStatus == storage.StatusDown || writerStatus == storage.StatusDown {
		health.Status = storage.StatusDown
	}

	return health
}

//nolint:dupl // getWriterDetails provides the publishing details for current google publishers.
func (g *googleClient) getWriterDetails() (status string, details map[string]any) {
	const contextTimeoutDuration = 50

	status = storage.StatusUp

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeoutDuration*time.Millisecond)
	defer cancel()

	it := g.client.Topics(ctx)

	details = make(map[string]any)

	for {
		topic, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			status = storage.StatusDown

			break
		}

		if topic != nil {
			details[topic.ID()] = topic
		}
	}

	return status, details
}

//nolint:dupl // getReaderDetails provides the subscription details for current google subscriptions.
func (g *googleClient) getReaderDetails() (status string, details map[string]any) {
	const contextTimeoutDuration = 50

	status = storage.StatusUp

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeoutDuration*time.Millisecond)
	defer cancel()

	subIt := g.client.Subscriptions(ctx)

	details = make(map[string]any)

	for {
		subscription, err := subIt.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			status = storage.StatusDown

			break
		}

		if subscription != nil {
			details[subscription.ID()] = subscription
		}
	}

	return status, details
}
