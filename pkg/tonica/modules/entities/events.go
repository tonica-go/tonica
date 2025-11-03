package entities

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	eventTypeRecordCreated = "entity.record.created"
	eventTypeRecordUpdated = "entity.record.updated"
	eventTypeRecordDeleted = "entity.record.deleted"
	eventTypeRecordIndexed = "entity.record.indexed"
)

func legacyRecordStreamID(entityID, recordID string) string {
	return fmt.Sprintf("entity:%s:%s", entityID, recordID)
}

func legacyIndexStreamID(entityID string) string {
	return fmt.Sprintf("entity:%s:index", entityID)
}

type eventMetadata struct {
	Entity    string    `json:"entity"`
	RecordID  string    `json:"record_id,omitempty"`
	ActorID   string    `json:"actor_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type recordPayload struct {
	Data map[string]any `json:"data"`
}

type indexPayload struct {
	RecordID string `json:"record_id"`
	Deleted  bool   `json:"deleted"`
}

func recordStreamID(entityID, recordID string) string {
	if recordID == "" {
		return fmt.Sprintf("entity:%s:%s", entityID, uuid.NewString())
	}
	return recordID
}

func indexStreamID(entityID string) string {
	if len(entityID) <= 32 {
		return fmt.Sprintf("idx:%s", entityID)
	}
	sum := sha1.Sum([]byte(entityID))
	return fmt.Sprintf("idx:%x", sum[:8])
}

func aggregateType(entityID string) string {
	return fmt.Sprintf("entity:%s", entityID)
}

func decodeEventMetadata(data []byte) (eventMetadata, error) {
	if len(data) == 0 {
		return eventMetadata{}, fmt.Errorf("missing metadata")
	}
	var meta eventMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return eventMetadata{}, fmt.Errorf("decode metadata: %w", err)
	}
	if meta.Timestamp.IsZero() {
		meta.Timestamp = time.Now().UTC()
	}
	return meta, nil
}

func decodeRecordPayload(data []byte) (recordPayload, error) {
	if len(data) == 0 {
		return recordPayload{}, nil
	}
	var payload recordPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return recordPayload{}, fmt.Errorf("decode record payload: %w", err)
	}
	if payload.Data == nil {
		payload.Data = make(map[string]any)
	}
	return payload, nil
}

func decodeIndexPayload(data []byte) (indexPayload, error) {
	if len(data) == 0 {
		return indexPayload{}, fmt.Errorf("missing index payload")
	}
	var payload indexPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return indexPayload{}, fmt.Errorf("decode index payload: %w", err)
	}
	return payload, nil
}
