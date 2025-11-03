package entities

import (
	pb "github.com/tonica-go/tonica/pkg/tonica/proto/entities"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// recordToProto converts a domain Record to protobuf Record.
func recordToProto(r Record) *pb.Record {
	data, _ := structpb.NewStruct(r.Data)

	return &pb.Record{
		Entity: r.Entity,
		Id:     r.ID,
		Data:   data,
		Metadata: &pb.RecordMetadata{
			Id:        r.ID,
			CreatedAt: timestamppb.New(r.CreatedAt),
			UpdatedAt: timestamppb.New(r.UpdatedAt),
			CreatedBy: r.CreatedBy,
			UpdatedBy: r.UpdatedBy,
		},
	}
}

// historyToProto converts a domain HistoryEntry to protobuf RecordHistoryEntry.
func historyToProto(h HistoryEntry) *pb.RecordHistoryEntry {
	data, _ := structpb.NewStruct(h.Data)

	return &pb.RecordHistoryEntry{
		Version:    h.Version,
		EventType:  h.EventType,
		OccurredAt: timestamppb.New(h.Timestamp),
		Actor:      h.Actor,
		Data:       data,
		RawPayload: string(h.RawPayload),
		Deleted:    h.Deleted,
	}
}

// pivotToProto converts a domain PivotResult to protobuf PivotResponse.
func pivotToProto(r PivotResult) *pb.PivotResponse {
	entries := make([]*pb.PivotEntry, 0, len(r.Entries))
	for _, e := range r.Entries {
		entries = append(entries, &pb.PivotEntry{
			RowKey:    e.RowKey,
			ColumnKey: e.ColumnKey,
			Value:     e.Value,
		})
	}

	totals := &pb.PivotTotals{
		Row:    r.RowTotals,
		Column: r.ColumnTotals,
		Grand:  r.GrandTotal,
	}

	rowField := r.RowField.ID
	columnField := ""
	if r.ColumnField != nil {
		columnField = r.ColumnField.ID
	}

	return &pb.PivotResponse{
		RowField:    rowField,
		ColumnField: columnField,
		Entries:     entries,
		Totals:      totals,
	}
}
