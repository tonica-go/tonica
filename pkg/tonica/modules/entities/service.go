package entities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	entityPb "github.com/tonica-go/tonica/pkg/tonica/proto/entities"

	"github.com/tonica-go/tonica/pkg/tonica/modules/eventstore"
)

// SearchIndexer provides search indexing capabilities for entities.
type SearchIndexer interface {
	IndexDocument(ctx context.Context, doc any) error
	DeleteDocument(ctx context.Context, entityType, entityID string) error
}

// Service provides metadata-driven CRUD operations backed by the event store.
type Service struct {
	defs      map[string]Definition
	store     eventstore.Store
	providers map[string]Provider
	indexer   SearchIndexer
}

// Record represents a materialized entity instance.
type Record struct {
	Entity    string
	ID        string
	Data      map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy string
	UpdatedBy string
	Version   int64
	Deleted   bool
}

// ListOptions contains optional filters for ListRecords.
type ListOptions struct {
	Filters   []Filter
	SortField string
	SortDir   entityPb.SortDirection
	PageSize  int
	PageToken string
	Search    string
}

// HistoryOptions control pagination for record history.
type HistoryOptions struct {
	PageSize  int
	PageToken string
}

// Filter captures a declarative filter expression.
type Filter struct {
	FieldID  string
	Operator entityPb.FilterOperator
	Value    any
}

// HistoryEntry captures a snapshot of a record at a specific version.
type HistoryEntry struct {
	Version    int64
	EventType  string
	Timestamp  time.Time
	Actor      string
	Data       map[string]any
	RawPayload []byte
	Deleted    bool
}

// PivotOptions controls pivot aggregation for records.
type PivotOptions struct {
	RowField    string
	ColumnField string
	Filters     []Filter
}

// PivotEntry captures a single aggregated cell.
type PivotEntry struct {
	RowKey    string
	ColumnKey string
	Value     float64
}

// PivotResult contains aggregated totals for a pivot query.
type PivotResult struct {
	RowField     FieldDefinition
	ColumnField  *FieldDefinition
	Entries      []PivotEntry
	RowTotals    map[string]float64
	ColumnTotals map[string]float64
	GrandTotal   float64
}

// Provider allows overriding CRUD behaviour for a specific entity.
type Provider interface {
	List(ctx context.Context, def Definition, opts ListOptions) ([]Record, string, error)
	Get(ctx context.Context, def Definition, id string) (Record, error)
	Create(ctx context.Context, def Definition, data map[string]any) (Record, error)
	Update(ctx context.Context, def Definition, id string, data map[string]any) (Record, error)
	Delete(ctx context.Context, def Definition, id string) error
}

// NewService constructs Service from embedded definitions.
func NewService(store eventstore.Store) (*Service, error) {
	defs, err := LoadDefinitions()
	if err != nil {
		return nil, err
	}
	return &Service{
		defs:      defs,
		store:     store,
		providers: make(map[string]Provider),
	}, nil
}

// RegisterProvider installs an override for the given entity id.
func (s *Service) RegisterProvider(entityID string, provider Provider) {
	if entityID == "" || provider == nil {
		return
	}
	if s.providers == nil {
		s.providers = make(map[string]Provider)
	}
	s.providers[strings.ToLower(entityID)] = provider
}

// AttachSearchIndexer configures an optional search indexer.
func (s *Service) AttachSearchIndexer(indexer SearchIndexer) {
	s.indexer = indexer
}

// ListEntities returns all registered entity definitions.
func (s *Service) ListEntities() []Definition {
	defs := make([]Definition, 0, len(s.defs))
	for _, def := range s.defs {
		defs = append(defs, def)
	}
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].ID < defs[j].ID
	})
	return defs
}

// Definition returns metadata for a specific entity.
func (s *Service) Definition(entityID string) (Definition, error) {
	def, ok := s.defs[strings.ToLower(entityID)]
	if !ok {
		return Definition{}, fmt.Errorf("%w: %s", ErrUnknownEntity, entityID)
	}
	return def, nil
}

func (s *Service) providerFor(entityID string) (Provider, bool) {
	if s.providers == nil {
		return nil, false
	}
	provider, ok := s.providers[strings.ToLower(entityID)]
	return provider, ok
}

func actorIDFromContext(ctx context.Context) (string, error) {
	// Try to get identity from context
	value := ctx.Value("identity")
	if value == nil {
		return "", ErrUnauthenticated
	}

	ident, ok := value.(map[string]interface{})
	if !ok || ident == nil || strings.TrimSpace(ident["id"].(string)) == "" {
		return "", ErrUnauthenticated
	}

	return ident["id"].(string), nil
}

// ListRecords materializes records for an entity using metadata filters.
func (s *Service) ListRecords(ctx context.Context, entityID string, opts ListOptions) ([]Record, string, error) {
	def, err := s.Definition(entityID)
	if err != nil {
		return nil, "", err
	}

	normFilters, err := s.normalizeFilters(def, opts.Filters)
	if err != nil {
		return nil, "", err
	}

	localFilters := make([]normalizedFilter, 0, len(normFilters))
	nestedFilters := make([]normalizedFilter, 0)
	for _, nf := range normFilters {
		if nf.isNested() {
			nestedFilters = append(nestedFilters, nf)
			continue
		}
		localFilters = append(localFilters, nf)
	}

	generatedFilters, emptyResult, err := s.buildNestedFilters(ctx, nestedFilters)
	if err != nil {
		return nil, "", err
	}
	if len(nestedFilters) > 0 && emptyResult {
		return []Record{}, "", nil
	}

	sanitizedFilters := make([]Filter, 0, len(localFilters)+len(generatedFilters))
	for _, nf := range localFilters {
		sanitizedFilters = append(sanitizedFilters, Filter{
			FieldID:  nf.targetField().ID,
			Operator: nf.Operator,
			Value:    nf.Value,
		})
	}
	sanitizedFilters = append(sanitizedFilters, generatedFilters...)
	opts.Filters = sanitizedFilters

	if len(generatedFilters) > 0 {
		normFilters, err = s.normalizeFilters(def, sanitizedFilters)
		if err != nil {
			return nil, "", err
		}
	} else {
		normFilters = localFilters
	}

	sortField := strings.TrimSpace(opts.SortField)
	if sortField == "" {
		sortField = def.PrimaryKey
	}
	if _, ok := def.Field(sortField); !ok && sortField != def.PrimaryKey {
		return nil, "", fmt.Errorf("%w: unknown sort field %s", ErrInvalidSort, sortField)
	}
	opts.SortField = sortField

	sortDir := opts.SortDir
	if sortDir == entityPb.SortDirection_SORT_DIRECTION_UNSPECIFIED {
		sortDir = entityPb.SortDirection_SORT_DIRECTION_ASC
	}
	opts.SortDir = sortDir
	opts.PageSize = clampPageSize(opts.PageSize)
	opts.Search = strings.TrimSpace(opts.Search)

	if provider, ok := s.providerFor(entityID); ok {
		return provider.List(ctx, def, opts)
	}

	return s.listRecordsDefault(ctx, def, normFilters, opts)
}

func (s *Service) listRecordsDefault(ctx context.Context, def Definition, filters []normalizedFilter, opts ListOptions) ([]Record, string, error) {
	indexEntries, err := s.loadIndex(ctx, def.ID)
	if err != nil {
		return nil, "", err
	}

	records := make([]Record, 0, len(indexEntries))
	for _, entry := range indexEntries {
		record, err := s.loadRecord(ctx, def, entry.RecordID)
		if errors.Is(err, ErrRecordNotFound) || errors.Is(err, ErrRecordDeleted) {
			continue
		}
		if err != nil {
			return nil, "", err
		}
		records = append(records, record)
	}

	records = applySearch(def, records, opts.Search)
	records = applyFilters(records, filters)
	sortRecords(records, opts.SortField, opts.SortDir)

	offset, err := parsePageToken(opts.PageToken)
	if err != nil {
		return nil, "", err
	}
	if offset >= len(records) {
		return []Record{}, "", nil
	}

	end := offset + opts.PageSize
	if end > len(records) {
		end = len(records)
	}
	page := records[offset:end]

	var nextToken string
	if end < len(records) {
		nextToken = strconv.Itoa(end)
	}
	return page, nextToken, nil
}

// GetRecord returns a single record by id.
func (s *Service) GetRecord(ctx context.Context, entityID, recordID string) (Record, error) {
	def, err := s.Definition(entityID)
	if err != nil {
		return Record{}, err
	}

	if provider, ok := s.providerFor(entityID); ok {
		return provider.Get(ctx, def, recordID)
	}

	return s.getRecordDefault(ctx, def, recordID)
}

// RecordHistory returns the timeline of changes for a record.
func (s *Service) RecordHistory(ctx context.Context, entityID, recordID string, opts HistoryOptions) ([]HistoryEntry, string, error) {
	def, err := s.Definition(entityID)
	if err != nil {
		return nil, "", err
	}

	streamID := recordStreamID(def.ID, recordID)
	events, err := s.store.Load(ctx, streamID, 0)
	if err != nil {
		return nil, "", err
	}
	if len(events) == 0 {
		legacyID := legacyRecordStreamID(def.ID, recordID)
		legacyEvents, legacyErr := s.store.Load(ctx, legacyID, 0)
		if legacyErr != nil {
			return nil, "", legacyErr
		}
		events = legacyEvents
		if len(events) == 0 {
			return nil, "", ErrRecordNotFound
		}
	}

	state := make(map[string]any)
	entries := make([]HistoryEntry, 0, len(events))

	for _, evt := range events {
		meta, err := decodeEventMetadata(evt.Metadata)
		if err != nil {
			return nil, "", err
		}

		entry := HistoryEntry{
			Version:    evt.Version,
			EventType:  evt.Type,
			Timestamp:  meta.Timestamp,
			Actor:      meta.ActorID,
			RawPayload: append([]byte(nil), evt.Payload...),
		}

		switch evt.Type {
		case eventTypeRecordCreated:
			payload, err := decodeRecordPayload(evt.Payload)
			if err != nil {
				return nil, "", err
			}
			state = cloneMap(payload.Data)
			entry.Data = snapshotForHistory(def, state)
		case eventTypeRecordUpdated:
			payload, err := decodeRecordPayload(evt.Payload)
			if err != nil {
				return nil, "", err
			}
			if state == nil {
				state = make(map[string]any)
			}
			for key, value := range payload.Data {
				if value == nil {
					delete(state, key)
					continue
				}
				state[key] = value
			}
			entry.Data = snapshotForHistory(def, state)
		case eventTypeRecordDeleted:
			entry.Data = snapshotForHistory(def, state)
			entry.Deleted = true
		default:
			continue
		}

		entries = append(entries, entry)
	}

	// Sort newest first for presentation.
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	pageSize := clampPageSize(opts.PageSize)
	offset, err := parsePageToken(opts.PageToken)
	if err != nil {
		return nil, "", err
	}
	if offset >= len(entries) {
		return []HistoryEntry{}, "", nil
	}

	end := offset + pageSize
	if end > len(entries) {
		end = len(entries)
	}
	page := entries[offset:end]

	var nextToken string
	if end < len(entries) {
		nextToken = strconv.Itoa(end)
	}

	return page, nextToken, nil
}

// PivotRecords aggregates records for simple pivot reporting.
func (s *Service) PivotRecords(ctx context.Context, entityID string, opts PivotOptions) (PivotResult, error) {
	def, err := s.Definition(entityID)
	if err != nil {
		return PivotResult{}, err
	}

	rowFieldID := strings.TrimSpace(opts.RowField)
	if rowFieldID == "" {
		return PivotResult{}, fmt.Errorf("%w: row field is required", ErrInvalidFilter)
	}
	rowField, ok := def.Field(rowFieldID)
	if !ok {
		return PivotResult{}, fmt.Errorf("%w: row field %s", ErrInvalidFilter, rowFieldID)
	}

	var columnFieldPtr *FieldDefinition
	columnFieldID := strings.TrimSpace(opts.ColumnField)
	if columnFieldID != "" {
		columnField, ok := def.Field(columnFieldID)
		if !ok {
			return PivotResult{}, fmt.Errorf("%w: column field %s", ErrInvalidFilter, columnFieldID)
		}
		columnFieldPtr = &columnField
	}

	listOpts := ListOptions{
		Filters:   opts.Filters,
		SortField: def.PrimaryKey,
		SortDir:   entityPb.SortDirection_SORT_DIRECTION_ASC,
		PageSize:  200,
	}

	pageToken := ""
	counts := make(map[string]map[string]float64)
	rowTotals := make(map[string]float64)
	columnTotals := make(map[string]float64)
	var grandTotal float64

	for {
		listOpts.PageToken = pageToken
		records, next, err := s.ListRecords(ctx, entityID, listOpts)
		if err != nil {
			return PivotResult{}, err
		}

		for _, record := range records {
			rowKey := normalizePivotKey(asString(record.Data[rowField.ID]))
			columnKey := ""
			if columnFieldPtr != nil {
				columnKey = normalizePivotKey(asString(record.Data[columnFieldPtr.ID]))
			}

			if _, exists := counts[rowKey]; !exists {
				counts[rowKey] = make(map[string]float64)
			}
			counts[rowKey][columnKey]++
			rowTotals[rowKey]++
			columnTotals[columnKey]++
			grandTotal++
		}

		if next == "" {
			break
		}
		if next == pageToken {
			break
		}
		pageToken = next
	}

	entries := make([]PivotEntry, 0)
	rowKeys := make([]string, 0, len(counts))
	for key := range counts {
		rowKeys = append(rowKeys, key)
	}
	sort.Strings(rowKeys)

	if columnFieldPtr != nil {
		columnKeySet := make(map[string]struct{})
		for _, columnMap := range counts {
			for key := range columnMap {
				columnKeySet[key] = struct{}{}
			}
		}
		columnKeys := make([]string, 0, len(columnKeySet))
		for key := range columnKeySet {
			columnKeys = append(columnKeys, key)
		}
		sort.Strings(columnKeys)

		for _, rowKey := range rowKeys {
			for _, columnKey := range columnKeys {
				value := counts[rowKey][columnKey]
				if value == 0 {
					continue
				}
				entries = append(entries, PivotEntry{
					RowKey:    rowKey,
					ColumnKey: columnKey,
					Value:     value,
				})
			}
		}
	} else {
		for _, rowKey := range rowKeys {
			value := rowTotals[rowKey]
			if value == 0 {
				continue
			}
			entries = append(entries, PivotEntry{
				RowKey: rowKey,
				Value:  value,
			})
		}
		columnTotals = map[string]float64{}
	}

	return PivotResult{
		RowField:     rowField,
		ColumnField:  columnFieldPtr,
		Entries:      entries,
		RowTotals:    rowTotals,
		ColumnTotals: columnTotals,
		GrandTotal:   grandTotal,
	}, nil
}

func (s *Service) getRecordDefault(ctx context.Context, def Definition, recordID string) (Record, error) {
	record, err := s.loadRecord(ctx, def, recordID)
	if err != nil {
		return Record{}, err
	}
	if record.Deleted {
		return Record{}, fmt.Errorf("%w: %s/%s", ErrRecordDeleted, def.ID, recordID)
	}
	return record, nil
}

// CreateRecord appends events for a new record.
func (s *Service) CreateRecord(ctx context.Context, entityID string, payload map[string]any) (Record, error) {
	def, err := s.Definition(entityID)
	if err != nil {
		return Record{}, err
	}

	data, err := sanitizePayload(def, payload, false)
	if err != nil {
		return Record{}, err
	}

	if provider, ok := s.providerFor(entityID); ok {
		return provider.Create(ctx, def, data)
	}

	return s.createRecordDefault(ctx, def, data)
}

func (s *Service) createRecordDefault(ctx context.Context, def Definition, data map[string]any) (Record, error) {
	actorID, err := actorIDFromContext(ctx)
	if err != nil {
		return Record{}, err
	}

	recordID := asString(data[def.PrimaryKey])
	if recordID == "" {
		recordID = uuid.NewString()
		data[def.PrimaryKey] = recordID
	}

	if _, err := s.loadRecord(ctx, def, recordID); err == nil {
		return Record{}, fmt.Errorf("record %s already exists", recordID)
	} else if !errors.Is(err, ErrRecordNotFound) {
		return Record{}, err
	}

	now := time.Now().UTC()
	data["createdBy"] = actorID
	data["updatedBy"] = actorID

	meta := eventMetadata{
		Entity:    def.ID,
		RecordID:  recordID,
		ActorID:   actorID,
		Timestamp: now,
	}

	if err := s.appendRecordEvent(ctx, def.ID, recordID, -1, eventTypeRecordCreated, data, meta); err != nil {
		return Record{}, err
	}

	if err := s.appendIndexEvent(ctx, def.ID, recordID, false, meta); err != nil {
		return Record{}, err
	}

	record := Record{
		Entity:    def.ID,
		ID:        recordID,
		Data:      cloneMap(data),
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: actorID,
		UpdatedBy: actorID,
		Version:   1,
	}

	//s.emitEntityNotification(ctx, eventTypeRecordCreated, def, record, meta)
	s.indexRecord(ctx, record)
	return record, nil
}

// UpdateRecord appends an update event and returns latest state.
func (s *Service) UpdateRecord(ctx context.Context, entityID, recordID string, payload map[string]any) (Record, error) {
	def, err := s.Definition(entityID)
	if err != nil {
		return Record{}, err
	}

	data, err := sanitizePayload(def, payload, true)
	if err != nil {
		return Record{}, err
	}
	delete(data, def.PrimaryKey)
	if len(data) == 0 {
		if provider, ok := s.providerFor(entityID); ok {
			return provider.Get(ctx, def, recordID)
		}
		return s.getRecordDefault(ctx, def, recordID)
	}

	if provider, ok := s.providerFor(entityID); ok {
		return provider.Update(ctx, def, recordID, data)
	}

	return s.updateRecordDefault(ctx, def, recordID, data)
}

func (s *Service) updateRecordDefault(ctx context.Context, def Definition, recordID string, data map[string]any) (Record, error) {
	actorID, err := actorIDFromContext(ctx)
	if err != nil {
		return Record{}, err
	}

	current, err := s.loadRecord(ctx, def, recordID)
	if err != nil {
		return Record{}, err
	}
	if current.Deleted {
		return Record{}, fmt.Errorf("%w: %s/%s", ErrRecordDeleted, def.ID, recordID)
	}

	now := time.Now().UTC()
	data["updatedBy"] = actorID

	meta := eventMetadata{
		Entity:    def.ID,
		RecordID:  recordID,
		ActorID:   actorID,
		Timestamp: now,
	}

	if err := s.appendRecordEvent(ctx, def.ID, recordID, current.Version, eventTypeRecordUpdated, data, meta); err != nil {
		return Record{}, err
	}

	if err := s.appendIndexEvent(ctx, def.ID, recordID, false, meta); err != nil {
		return Record{}, err
	}

	for key, value := range data {
		if value == nil {
			delete(current.Data, key)
			continue
		}
		current.Data[key] = value
	}
	current.UpdatedAt = now
	if actorID != "" {
		current.UpdatedBy = actorID
	}
	current.Version++

	//s.emitEntityNotification(ctx, eventTypeRecordUpdated, def, current, meta)
	s.indexRecord(ctx, current)
	return current, nil
}

// DeleteRecord appends a tombstone event and marks index entry as deleted.
func (s *Service) DeleteRecord(ctx context.Context, entityID, recordID string) error {
	def, err := s.Definition(entityID)
	if err != nil {
		return err
	}

	if provider, ok := s.providerFor(entityID); ok {
		return provider.Delete(ctx, def, recordID)
	}

	return s.deleteRecordDefault(ctx, def, recordID)
}

func (s *Service) deleteRecordDefault(ctx context.Context, def Definition, recordID string) error {
	actorID, err := actorIDFromContext(ctx)
	if err != nil {
		return err
	}

	current, err := s.loadRecord(ctx, def, recordID)
	if err != nil {
		return err
	}
	if current.Deleted {
		return nil
	}

	now := time.Now().UTC()
	meta := eventMetadata{
		Entity:    def.ID,
		RecordID:  recordID,
		ActorID:   actorID,
		Timestamp: now,
	}

	if err := s.appendRecordEvent(ctx, def.ID, recordID, current.Version, eventTypeRecordDeleted, nil, meta); err != nil {
		return err
	}
	if err := s.appendIndexEvent(ctx, def.ID, recordID, true, meta); err != nil {
		return err
	}

	//s.emitEntityNotification(ctx, eventTypeRecordDeleted, def, current, meta)
	s.deleteRecordFromIndex(ctx, def.ID, recordID)
	return nil
}

func (s *Service) loadRecord(ctx context.Context, def Definition, recordID string) (Record, error) {
	streamID := recordStreamID(def.ID, recordID)
	events, err := s.store.Load(ctx, streamID, 0)
	if err != nil {
		return Record{}, err
	}
	if len(events) == 0 {
		legacyID := legacyRecordStreamID(def.ID, recordID)
		legacyEvents, legacyErr := s.store.Load(ctx, legacyID, 0)
		if legacyErr != nil {
			return Record{}, legacyErr
		}
		events = legacyEvents
		if len(events) == 0 {
			return Record{}, ErrRecordNotFound
		}
	}

	state := Record{
		Entity: def.ID,
		ID:     recordID,
		Data:   make(map[string]any),
	}

	for _, evt := range events {
		meta, err := decodeEventMetadata(evt.Metadata)
		if err != nil {
			return Record{}, err
		}

		switch evt.Type {
		case eventTypeRecordCreated:
			payload, err := decodeRecordPayload(evt.Payload)
			if err != nil {
				return Record{}, err
			}
			state.Data = cloneMap(payload.Data)
			state.CreatedAt = meta.Timestamp
			state.UpdatedAt = meta.Timestamp
			state.CreatedBy = meta.ActorID
			state.UpdatedBy = meta.ActorID
			if meta.ActorID != "" {
				if _, exists := state.Data["createdBy"]; !exists {
					state.Data["createdBy"] = meta.ActorID
				}
				if _, exists := state.Data["updatedBy"]; !exists {
					state.Data["updatedBy"] = meta.ActorID
				}
			}
			state.Version = evt.Version
			state.Deleted = false
		case eventTypeRecordUpdated:
			payload, err := decodeRecordPayload(evt.Payload)
			if err != nil {
				return Record{}, err
			}
			for key, value := range payload.Data {
				if value == nil {
					delete(state.Data, key)
				} else {
					state.Data[key] = value
				}
			}
			state.UpdatedAt = meta.Timestamp
			if meta.ActorID != "" {
				state.UpdatedBy = meta.ActorID
				state.Data["updatedBy"] = meta.ActorID
			}
			state.Version = evt.Version
		case eventTypeRecordDeleted:
			state.Deleted = true
			state.UpdatedAt = meta.Timestamp
			if meta.ActorID != "" {
				state.UpdatedBy = meta.ActorID
			}
			state.Version = evt.Version
		default:
			continue
		}
	}

	return state, nil
}

type indexState struct {
	RecordID string
	Deleted  bool
}

func (s *Service) loadIndex(ctx context.Context, entityID string) ([]indexState, error) {
	streamID := indexStreamID(entityID)
	events, err := s.store.Load(ctx, streamID, 0)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		legacyID := legacyIndexStreamID(entityID)
		legacyEvents, legacyErr := s.store.Load(ctx, legacyID, 0)
		if legacyErr != nil {
			return nil, legacyErr
		}
		events = legacyEvents
		if len(events) == 0 {
			return []indexState{}, nil
		}
	}

	state := make(map[string]indexState)
	for _, evt := range events {
		if evt.Type != eventTypeRecordIndexed {
			continue
		}
		payload, err := decodeIndexPayload(evt.Payload)
		if err != nil {
			return nil, err
		}
		state[payload.RecordID] = indexState{
			RecordID: payload.RecordID,
			Deleted:  payload.Deleted,
		}
	}

	result := make([]indexState, 0, len(state))
	for _, entry := range state {
		if entry.Deleted {
			continue
		}
		result = append(result, entry)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].RecordID < result[j].RecordID
	})
	return result, nil
}

func (s *Service) appendRecordEvent(ctx context.Context, entityID, recordID string, expectedVersion int64, eventType string, data map[string]any, meta eventMetadata) error {
	payload := recordPayload{Data: data}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	event := eventstore.Event{
		AggregateType: aggregateType(entityID),
		Type:          eventType,
		Payload:       payloadBytes,
		Metadata:      metaBytes,
	}
	streamID := recordStreamID(entityID, recordID)
	if err := s.store.Append(ctx, streamID, expectedVersion, []eventstore.Event{event}); err != nil {
		// Preserve concurrency conflict errors for retry logic
		if errors.Is(err, eventstore.ErrConcurrencyConflict) {
			return fmt.Errorf("%w: %s/%s at version %d", eventstore.ErrConcurrencyConflict, entityID, recordID, expectedVersion)
		}
		return err
	}
	legacyID := legacyRecordStreamID(entityID, recordID)
	if streamID != legacyID {
		_ = s.store.Append(ctx, legacyID, -1, []eventstore.Event{event})
	}
	return nil
}

func (s *Service) appendIndexEvent(ctx context.Context, entityID, recordID string, deleted bool, meta eventMetadata) error {
	payload := indexPayload{
		RecordID: recordID,
		Deleted:  deleted,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal index payload: %w", err)
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	event := eventstore.Event{
		AggregateType: aggregateType(entityID),
		Type:          eventTypeRecordIndexed,
		Payload:       payloadBytes,
		Metadata:      metaBytes,
	}
	if err := s.store.Append(ctx, indexStreamID(entityID), -1, []eventstore.Event{event}); err != nil {
		return err
	}
	legacyID := legacyIndexStreamID(entityID)
	if legacyID != indexStreamID(entityID) {
		_ = s.store.Append(ctx, legacyID, -1, []eventstore.Event{event})
	}
	return nil
}

//func (s *Service) emitEntityNotification(ctx context.Context, eventType string, def Definition, record Record, meta eventMetadata) {
//	if s.notifier == nil {
//		return
//	}
//	event := notifications.EntityEvent{
//		EntityID:  def.ID,
//		Action:    eventType,
//		RecordID:  record.ID,
//		ActorID:   meta.ActorID,
//		Data:      cloneMap(record.Data),
//		Version:   int(record.Version),
//		Timestamp: meta.Timestamp,
//	}
//	if _, err := s.notifier.PublishEntityEvent(ctx, event); err != nil {
//		s.logger.Warn("notify entity event failed", zap.String("entity", def.ID), zap.String("record_id", record.ID), zap.Error(err))
//		return
//	}
//
//}

func (s *Service) normalizeFilters(def Definition, filters []Filter) ([]normalizedFilter, error) {
	out := make([]normalizedFilter, 0, len(filters))
	for _, filter := range filters {
		fieldPath := strings.TrimSpace(filter.FieldID)
		if fieldPath == "" {
			return nil, fmt.Errorf("%w: empty filter field", ErrInvalidFilter)
		}

		operator := filter.Operator
		if operator == entityPb.FilterOperator_FILTER_OPERATOR_UNSPECIFIED {
			operator = entityPb.FilterOperator_FILTER_OPERATOR_EQ
		}

		segments := strings.Split(fieldPath, ".")
		steps, err := s.resolveRelationshipPath(def, segments)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidFilter, err)
		}

		targetField := steps[len(steps)-1].Field
		if len(targetField.FilterOperators) > 0 && !targetField.Supports(operator) {
			return nil, fmt.Errorf("%w: %s does not support operator %s", ErrInvalidFilter, fieldPath, operator.String())
		}

		var (
			value   any
			convErr error
		)
		if operator == entityPb.FilterOperator_FILTER_OPERATOR_IN {
			value, convErr = coerceInValue(targetField.Type, filter.Value)
		} else {
			value, convErr = coerceValue(targetField.Type, filter.Value)
		}
		if convErr != nil {
			return nil, fmt.Errorf("%w: %s %v", ErrInvalidFilter, fieldPath, convErr)
		}
		out = append(out, normalizedFilter{
			Steps:         steps,
			Operator:      operator,
			Value:         value,
			OriginalField: fieldPath,
		})
	}
	return out, nil
}

type normalizedFilter struct {
	Steps         []relationshipStep
	Operator      entityPb.FilterOperator
	Value         any
	OriginalField string
}

func (nf normalizedFilter) targetField() FieldDefinition {
	return nf.Steps[len(nf.Steps)-1].Field
}

func (nf normalizedFilter) isNested() bool {
	return len(nf.Steps) > 1
}

type relationshipStep struct {
	Entity    Definition
	Field     FieldDefinition
	Reference *ReferenceDefinition
}

func coerceInValue(fieldType entityPb.FieldType, value any) (any, error) {
	if value == nil {
		return []any{}, nil
	}

	var items []any

	switch v := value.(type) {
	case []any:
		items = v
	case []string:
		items = make([]any, len(v))
		for i, item := range v {
			items[i] = item
		}
	case string:
		raw := strings.TrimSpace(v)
		if raw == "" {
			return []any{}, nil
		}
		parts := strings.Split(raw, ",")
		items = make([]any, 0, len(parts))
		for _, part := range parts {
			items = append(items, strings.TrimSpace(part))
		}
	default:
		items = []any{v}
	}

	normalized := make([]any, 0, len(items))
	for _, item := range items {
		coerced, err := coerceValue(fieldType, item)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, coerced)
	}
	return normalized, nil
}

func (s *Service) buildNestedFilters(ctx context.Context, filters []normalizedFilter) ([]Filter, bool, error) {
	if len(filters) == 0 {
		return nil, false, nil
	}

	type nestedAggregate struct {
		field FieldDefinition
		ids   stringSet
	}

	accumulator := make(map[string]nestedAggregate)

	for _, filter := range filters {
		ids, err := s.evaluateNestedFilter(ctx, filter)
		if err != nil {
			return nil, false, err
		}
		if len(ids) == 0 {
			return nil, true, nil
		}

		fieldID := filter.Steps[0].Field.ID
		if existing, ok := accumulator[fieldID]; ok {
			intersection := intersectStringSets(existing.ids, ids)
			if len(intersection) == 0 {
				return nil, true, nil
			}
			existing.ids = intersection
			accumulator[fieldID] = existing
		} else {
			accumulator[fieldID] = nestedAggregate{
				field: filter.Steps[0].Field,
				ids:   ids,
			}
		}
	}

	result := make([]Filter, 0, len(accumulator))
	for fieldID, agg := range accumulator {
		values := agg.ids.toSlice()
		sort.Strings(values)
		operator := entityPb.FilterOperator_FILTER_OPERATOR_IN
		var value any = values
		if len(values) == 1 && !agg.field.Supports(entityPb.FilterOperator_FILTER_OPERATOR_IN) {
			operator = entityPb.FilterOperator_FILTER_OPERATOR_EQ
			value = values[0]
		}
		result = append(result, Filter{
			FieldID:  fieldID,
			Operator: operator,
			Value:    value,
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].FieldID < result[j].FieldID
	})

	return result, false, nil
}

func (s *Service) evaluateNestedFilter(ctx context.Context, filter normalizedFilter) (stringSet, error) {
	if len(filter.Steps) < 2 {
		return nil, nil
	}

	var nextValues stringSet

	for i := len(filter.Steps) - 1; i >= 1; i-- {
		step := filter.Steps[i]

		var stepFilters []Filter
		if i == len(filter.Steps)-1 {
			stepFilters = []Filter{{
				FieldID:  step.Field.ID,
				Operator: filter.Operator,
				Value:    filter.Value,
			}}
		} else {
			if len(nextValues) == 0 {
				return stringSet{}, nil
			}
			stepFilters = []Filter{{
				FieldID:  step.Field.ID,
				Operator: entityPb.FilterOperator_FILTER_OPERATOR_IN,
				Value:    nextValues.toSlice(),
			}}
		}

		records, err := s.collectAllRecords(ctx, step.Entity.ID, stepFilters)
		if err != nil {
			return nil, err
		}
		if len(records) == 0 {
			return stringSet{}, nil
		}

		prevStep := filter.Steps[i-1]
		if prevStep.Reference == nil {
			return nil, fmt.Errorf("field %s on %s is not a reference", prevStep.Field.ID, prevStep.Entity.ID)
		}

		targetField := strings.TrimSpace(prevStep.Reference.Field)
		if targetField == "" {
			targetField = step.Entity.PrimaryKey
		}

		nextValues = extractFieldValues(records, targetField, step.Entity.PrimaryKey)
		if len(nextValues) == 0 {
			return stringSet{}, nil
		}
	}

	return nextValues, nil
}

func (s *Service) collectAllRecords(ctx context.Context, entityID string, filters []Filter) ([]Record, error) {
	def, err := s.Definition(entityID)
	if err != nil {
		return nil, err
	}

	normFilters, err := s.normalizeFilters(def, filters)
	if err != nil {
		return nil, err
	}

	opts := ListOptions{
		Filters:  filters,
		PageSize: 200,
		SortDir:  entityPb.SortDirection_SORT_DIRECTION_ASC,
	}

	if opts.SortField == "" {
		opts.SortField = def.PrimaryKey
	}

	var (
		all       []Record
		pageToken string
	)

	for {
		opts.PageToken = pageToken

		var (
			batch     []Record
			nextToken string
			listErr   error
		)

		if provider, ok := s.providerFor(entityID); ok {
			batch, nextToken, listErr = provider.List(ctx, def, opts)
		} else {
			batch, nextToken, listErr = s.listRecordsDefault(ctx, def, normFilters, opts)
		}
		if listErr != nil {
			return nil, listErr
		}

		all = append(all, batch...)
		if nextToken == "" || nextToken == pageToken {
			break
		}
		pageToken = nextToken
	}

	return all, nil
}

func (s *Service) resolveRelationshipPath(def Definition, segments []string) ([]relationshipStep, error) {
	if len(segments) == 0 {
		return nil, fmt.Errorf("invalid empty field path for entity %s", def.ID)
	}

	current := def
	steps := make([]relationshipStep, 0, len(segments))

	for idx, rawSegment := range segments {
		segment := strings.TrimSpace(rawSegment)
		if segment == "" {
			return nil, fmt.Errorf("invalid empty path segment for entity %s", current.ID)
		}

		field, err := findFieldForSegment(current, segment)
		if err != nil {
			return nil, err
		}

		step := relationshipStep{
			Entity:    current,
			Field:     field,
			Reference: field.Reference,
		}
		steps = append(steps, step)

		if idx == len(segments)-1 {
			continue
		}

		if field.Reference == nil {
			return nil, fmt.Errorf("segment %q on %s is not a reference", segment, current.ID)
		}

		next, err := s.Definition(field.Reference.Entity)
		if err != nil {
			return nil, err
		}
		current = next
	}

	return steps, nil
}

func findFieldForSegment(def Definition, segment string) (FieldDefinition, error) {
	for _, field := range def.Fields {
		if matchesFieldSegment(field, segment) {
			return field, nil
		}
	}
	return FieldDefinition{}, fmt.Errorf("%w: %s.%s", ErrInvalidFilter, def.ID, segment)
}

func matchesFieldSegment(field FieldDefinition, segment string) bool {
	if strings.EqualFold(field.ID, segment) {
		return true
	}

	if field.Reference != nil {
		if alias := strings.TrimSpace(field.Reference.Metadata["entityId"]); alias != "" && strings.EqualFold(alias, segment) {
			return true
		}
	}

	if trimmed := trimFieldIdentifier(field.ID); trimmed != "" && strings.EqualFold(trimmed, segment) {
		return true
	}

	return false
}

func trimFieldIdentifier(id string) string {
	switch {
	case strings.HasSuffix(id, "Id"):
		return id[:len(id)-2]
	case strings.HasSuffix(id, "ID"):
		return id[:len(id)-2]
	case strings.HasSuffix(id, "_id"):
		return id[:len(id)-3]
	default:
		return ""
	}
}

func extractFieldValues(records []Record, fieldID string, primaryKey string) stringSet {
	result := make(stringSet, len(records))
	normalized := strings.TrimSpace(fieldID)

	for _, record := range records {
		if normalized != "" {
			if value, ok := record.Data[normalized]; ok {
				if str := asString(value); str != "" {
					result.add(str)
					continue
				}
			}
		}

		if record.ID != "" && (normalized == "" || strings.EqualFold(normalized, primaryKey) || strings.EqualFold(normalized, "id")) {
			result.add(record.ID)
		}
	}

	return result
}

type stringSet map[string]struct{}

func (s stringSet) add(value string) {
	if value == "" {
		return
	}
	s[value] = struct{}{}
}

func (s stringSet) toSlice() []string {
	result := make([]string, 0, len(s))
	for value := range s {
		result = append(result, value)
	}
	return result
}

func intersectStringSets(a, b stringSet) stringSet {
	if len(a) == 0 || len(b) == 0 {
		return stringSet{}
	}
	smaller, larger := a, b
	if len(b) < len(a) {
		smaller, larger = b, a
	}
	result := make(stringSet, len(smaller))
	for value := range smaller {
		if _, ok := larger[value]; ok {
			result[value] = struct{}{}
		}
	}
	return result
}

func applySearch(def Definition, records []Record, query string) []Record {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return records
	}

	fields := searchableFieldIDs(def)
	if len(fields) == 0 {
		return records
	}

	result := make([]Record, 0, len(records))
	for _, record := range records {
		if record.Deleted {
			continue
		}
		if matchesSearchFields(record.Data, fields, query) {
			result = append(result, record)
		}
	}
	return result
}

func searchableFieldIDs(def Definition) []string {
	ids := make([]string, 0, len(def.Fields))
	for _, field := range def.Fields {
		switch field.Type {
		case entityPb.FieldType_FIELD_TYPE_STRING,
			entityPb.FieldType_FIELD_TYPE_UUID,
			entityPb.FieldType_FIELD_TYPE_ENUM,
			entityPb.FieldType_FIELD_TYPE_DATETIME:
			ids = append(ids, field.ID)
		}
	}
	if def.PrimaryKey != "" {
		found := false
		for _, id := range ids {
			if id == def.PrimaryKey {
				found = true
				break
			}
		}
		if !found {
			ids = append(ids, def.PrimaryKey)
		}
	}
	return ids
}

func matchesSearchFields(data map[string]any, fields []string, query string) bool {
	for _, fieldID := range fields {
		value, ok := data[fieldID]
		if !ok {
			continue
		}
		if containsQuery(value, query) {
			return true
		}
	}
	return false
}

func containsQuery(value any, query string) bool {
	switch v := value.(type) {
	case string:
		return strings.Contains(strings.ToLower(v), query)
	case fmt.Stringer:
		return strings.Contains(strings.ToLower(v.String()), query)
	case []string:
		for _, item := range v {
			if strings.Contains(strings.ToLower(item), query) {
				return true
			}
		}
	case []any:
		for _, item := range v {
			if containsQuery(item, query) {
				return true
			}
		}
	}

	return strings.Contains(strings.ToLower(asString(value)), query)
}

func applyFilters(records []Record, filters []normalizedFilter) []Record {
	if len(filters) == 0 {
		return records
	}
	result := make([]Record, 0, len(records))
	for _, record := range records {
		if record.Deleted {
			continue
		}
		if matchesFilters(record.Data, filters) {
			result = append(result, record)
		}
	}
	return result
}

func matchesFilters(data map[string]any, filters []normalizedFilter) bool {
	for _, filter := range filters {
		field := filter.targetField()
		value, ok := data[field.ID]
		if !ok {
			return false
		}
		if !evaluateFilter(value, filter.Operator, filter.Value) {
			return false
		}
	}
	return true
}

func evaluateFilter(recordValue any, operator entityPb.FilterOperator, filterValue any) bool {
	switch operator {
	case entityPb.FilterOperator_FILTER_OPERATOR_EQ:
		return compareEquality(recordValue, filterValue)
	case entityPb.FilterOperator_FILTER_OPERATOR_NE:
		return !compareEquality(recordValue, filterValue)
	case entityPb.FilterOperator_FILTER_OPERATOR_CONTAINS:
		return compareContains(recordValue, filterValue)
	case entityPb.FilterOperator_FILTER_OPERATOR_IN:
		return compareIn(recordValue, filterValue)
	case entityPb.FilterOperator_FILTER_OPERATOR_GT,
		entityPb.FilterOperator_FILTER_OPERATOR_GTE,
		entityPb.FilterOperator_FILTER_OPERATOR_LT,
		entityPb.FilterOperator_FILTER_OPERATOR_LTE:
		return compareNumbers(recordValue, filterValue, operator)
	default:
		return true
	}
}

func compareEquality(a, b any) bool {
	switch va := a.(type) {
	case string:
		vb, ok := b.(string)
		return ok && strings.EqualFold(va, vb)
	case float64:
		vb, ok := b.(float64)
		return ok && va == vb
	case int:
		vb, ok := b.(int)
		return ok && va == vb
	case bool:
		vb, ok := b.(bool)
		return ok && va == vb
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func compareContains(a, b any) bool {
	switch va := a.(type) {
	case string:
		if vb, ok := b.(string); ok {
			return strings.Contains(strings.ToLower(va), strings.ToLower(vb))
		}
	case []any:
		for _, item := range va {
			if compareEquality(item, b) {
				return true
			}
		}
	case []string:
		if vb, ok := b.(string); ok {
			for _, item := range va {
				if strings.EqualFold(item, vb) {
					return true
				}
			}
		}
	}
	return false
}

func compareIn(a, b any) bool {
	switch vb := b.(type) {
	case []any:
		for _, candidate := range vb {
			if compareEquality(a, candidate) {
				return true
			}
		}
	case []string:
		for _, candidate := range vb {
			if compareEquality(a, candidate) {
				return true
			}
		}
	default:
		return compareEquality(a, b)
	}
	return false
}

func compareNumbers(a, b any, op entityPb.FilterOperator) bool {
	av, ok := toFloat64(a)
	if !ok {
		return false
	}
	bv, ok := toFloat64(b)
	if !ok {
		return false
	}
	switch op {
	case entityPb.FilterOperator_FILTER_OPERATOR_GT:
		return av > bv
	case entityPb.FilterOperator_FILTER_OPERATOR_GTE:
		return av >= bv
	case entityPb.FilterOperator_FILTER_OPERATOR_LT:
		return av < bv
	case entityPb.FilterOperator_FILTER_OPERATOR_LTE:
		return av <= bv
	default:
		return false
	}
}

func sortRecords(records []Record, field string, direction entityPb.SortDirection) {
	if len(records) <= 1 {
		return
	}
	ascending := direction != entityPb.SortDirection_SORT_DIRECTION_DESC
	sort.SliceStable(records, func(i, j int) bool {
		left := fetchSortable(records[i], field)
		right := fetchSortable(records[j], field)
		if ascending {
			return compareSortable(left, right) < 0
		}
		return compareSortable(left, right) > 0
	})
}

func fetchSortable(record Record, field string) any {
	if field == "" || strings.EqualFold(field, record.Entity) {
		return record.ID
	}
	if value, ok := record.Data[field]; ok {
		return value
	}
	return record.ID
}

func compareSortable(a, b any) int {
	switch va := a.(type) {
	case string:
		vb := fmt.Sprintf("%v", b)
		return strings.Compare(strings.ToLower(va), strings.ToLower(vb))
	case float64:
		vb, ok := toFloat64(b)
		if !ok {
			return 0
		}
		switch {
		case va < vb:
			return -1
		case va > vb:
			return 1
		default:
			return 0
		}
	case time.Time:
		vb, ok := b.(time.Time)
		if !ok {
			return 0
		}
		if va.Before(vb) {
			return -1
		}
		if va.After(vb) {
			return 1
		}
		return 0
	}
	return strings.Compare(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
}

func clampPageSize(size int) int {
	switch {
	case size <= 0:
		return 50
	case size > 200:
		return 200
	default:
		return size
	}
}

func parsePageToken(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(token)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid page token")
	}
	return value, nil
}

func sanitizePayload(def Definition, payload map[string]any, allowMissing bool) (map[string]any, error) {
	if payload == nil {
		return nil, fmt.Errorf("%w: empty payload", ErrInvalidPayload)
	}

	result := make(map[string]any, len(payload))
	var errs ValidationErrors

	for _, field := range def.Fields {
		value, exists := payload[field.ID]
		if !exists {
			if field.Required && !allowMissing {
				errs = append(errs, ValidationError{Field: field.ID, Message: "is required"})
			}
			continue
		}

		if isAuditField(field.ID) {
			continue
		}

		if value == nil {
			result[field.ID] = nil
			continue
		}

		normalized, err := coerceValue(field.Type, value)
		if err != nil {
			errs = append(errs, ValidationError{Field: field.ID, Message: err.Error()})
			continue
		}
		result[field.ID] = normalized
	}

	if len(errs) > 0 {
		return nil, errs
	}

	// Preserve unknown fields for forward compatibility but copy as-is.
	for key, value := range payload {
		if _, exists := result[key]; !exists && !isAuditField(key) {
			result[key] = value
		}
	}

	return result, nil
}

func isAuditField(fieldID string) bool {
	switch strings.ToLower(strings.TrimSpace(fieldID)) {
	case "createdby", "updatedby":
		return true
	default:
		return false
	}
}

func snapshotForHistory(def Definition, state map[string]any) map[string]any {
	if len(state) == 0 {
		return map[string]any{}
	}
	snapshot := make(map[string]any, len(def.Fields)+1)
	if def.PrimaryKey != "" {
		if value, ok := state[def.PrimaryKey]; ok {
			snapshot[def.PrimaryKey] = value
		}
	}
	for _, field := range def.Fields {
		if value, ok := state[field.ID]; ok {
			snapshot[field.ID] = value
		}
	}
	return snapshot
}

func normalizePivotKey(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "(none)"
	}
	return trimmed
}

func coerceValue(fieldType entityPb.FieldType, value any) (any, error) {
	switch fieldType {
	case entityPb.FieldType_FIELD_TYPE_STRING,
		entityPb.FieldType_FIELD_TYPE_UUID,
		entityPb.FieldType_FIELD_TYPE_ENUM,
		entityPb.FieldType_FIELD_TYPE_DATETIME:
		str := asString(value)
		if str == "" && value != "" {
			return nil, fmt.Errorf("expected string value")
		}
		return str, nil
	case entityPb.FieldType_FIELD_TYPE_BOOLEAN:
		switch v := value.(type) {
		case bool:
			return v, nil
		case string:
			lower := strings.ToLower(strings.TrimSpace(v))
			if lower == "true" || lower == "1" {
				return true, nil
			}
			if lower == "false" || lower == "0" {
				return false, nil
			}
		}
		return nil, fmt.Errorf("expected boolean value")
	case entityPb.FieldType_FIELD_TYPE_NUMBER:
		if number, ok := toFloat64(value); ok {
			return number, nil
		}
		return nil, fmt.Errorf("expected numeric value")
	case entityPb.FieldType_FIELD_TYPE_ARRAY:
		switch v := value.(type) {
		case []any:
			return v, nil
		case []string:
			items := make([]any, len(v))
			for i, item := range v {
				items[i] = item
			}
			return items, nil
		default:
			return []any{value}, nil
		}
	case entityPb.FieldType_FIELD_TYPE_OBJECT:
		switch v := value.(type) {
		case map[string]any:
			return v, nil
		default:
			return nil, fmt.Errorf("expected object value")
		}
	default:
		return value, nil
	}
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case []byte:
		return strings.TrimSpace(string(v))
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func toFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, false
		}
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint32:
		return float64(v), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

// indexRecord indexes a record in the search service if available.
func (s *Service) indexRecord(ctx context.Context, record Record) {
	if s.indexer == nil {
		return
	}
	if err := s.indexer.IndexDocument(ctx, record); err != nil {
		//s.logger.Warn("failed to index record",
		//	zap.String("entity", record.Entity),
		//	zap.String("record_id", record.ID),
		//	zap.Error(err))
		slog.Warn("Error indexing document", "error", err.Error())
	}
}

// deleteRecordFromIndex removes a record from the search index if available.
func (s *Service) deleteRecordFromIndex(ctx context.Context, entityID, recordID string) {
	if s.indexer == nil {
		return
	}
	if err := s.indexer.DeleteDocument(ctx, entityID, recordID); err != nil {
		//s.logger.Warn("failed to delete record from index",
		//	zap.String("entity", entityID),
		//	zap.String("record_id", recordID),
		//	zap.Error(err))
		slog.Warn("Error indexing document", "error", err.Error())
	}
}
