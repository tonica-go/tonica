package entities

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/tonica-go/tonica/pkg/tonica/service"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/tonica-go/tonica/pkg/tonica/proto/entities"

	"github.com/tonica-go/tonica/pkg/tonica/modules/eventstore"
)

// NewTonicaService creates a new tonica service for entities module.
func NewTonicaService(dsn, driver string) *service.Service {
	return service.NewService(
		service.WithName("entities"),
		service.WithGRPCAddr(":19002"),
		service.WithDB(dsn, driver),
		service.WithGRPC(registerGRPC),
		service.WithGateway(registerGateway),
	)
}

// registerGRPC registers the entities gRPC service.
func registerGRPC(grpcServer *grpc.Server, svc *service.Service) {
	bunDB := svc.GetDBClient()
	if bunDB == nil {
		panic("database not configured for entities service")
	}

	// Create event store from bun DB
	store, err := eventstore.NewFromBun(context.Background(), bunDB)
	if err != nil {
		panic(err)
	}

	// Create entities service
	entitySvc, err := NewService(store)
	if err != nil {
		panic(err)
	}

	// Register gRPC server
	handler := &grpcHandler{svc: entitySvc}
	pb.RegisterEntityServiceServer(grpcServer, handler)
}

// registerGateway registers the entities gateway.
func registerGateway(ctx context.Context, mux *runtime.ServeMux, target string, opts []grpc.DialOption) error {
	return pb.RegisterEntityServiceHandlerFromEndpoint(ctx, mux, target, opts)
}

// grpcHandler implements pb.EntityServiceServer
type grpcHandler struct {
	pb.UnimplementedEntityServiceServer
	svc *Service
}

func (h *grpcHandler) ListEntities(ctx context.Context, req *emptypb.Empty) (*pb.ListEntitiesResponse, error) {
	defs := h.svc.ListEntities()
	entities := make([]*pb.EntityDefinition, 0, len(defs))
	for _, def := range defs {
		entities = append(entities, def.ToProto())
	}
	return &pb.ListEntitiesResponse{Entities: entities}, nil
}

func (h *grpcHandler) GetEntity(ctx context.Context, req *pb.GetEntityRequest) (*pb.EntityDefinition, error) {
	def, err := h.svc.Definition(req.GetId())
	if err != nil {
		return nil, err
	}
	return def.ToProto(), nil
}

func (h *grpcHandler) ListRecords(ctx context.Context, req *pb.ListRecordsRequest) (*pb.ListRecordsResponse, error) {
	filters := make([]Filter, 0, len(req.GetFilters()))
	for _, f := range req.GetFilters() {
		filters = append(filters, Filter{
			FieldID:  f.GetField(),
			Operator: f.GetOperator(),
			Value:    f.GetValue().AsInterface(),
		})
	}

	opts := ListOptions{
		Filters:   filters,
		SortField: req.GetSortField(),
		SortDir:   req.GetSortDirection(),
		PageSize:  int(req.GetPageSize()),
		PageToken: req.GetPageToken(),
		Search:    req.GetSearch(),
	}

	records, nextToken, err := h.svc.ListRecords(ctx, req.GetEntity(), opts)
	if err != nil {
		return nil, err
	}

	pbRecords := make([]*pb.Record, 0, len(records))
	for _, r := range records {
		pbRecords = append(pbRecords, recordToProto(r))
	}

	return &pb.ListRecordsResponse{
		Records:       pbRecords,
		NextPageToken: nextToken,
	}, nil
}

func (h *grpcHandler) GetRecord(ctx context.Context, req *pb.GetRecordRequest) (*pb.Record, error) {
	record, err := h.svc.GetRecord(ctx, req.GetEntity(), req.GetId())
	if err != nil {
		return nil, err
	}
	return recordToProto(record), nil
}

func (h *grpcHandler) CreateRecord(ctx context.Context, req *pb.CreateRecordRequest) (*pb.Record, error) {
	data := req.GetData().AsMap()
	record, err := h.svc.CreateRecord(ctx, req.GetEntity(), data)
	if err != nil {
		return nil, err
	}
	return recordToProto(record), nil
}

func (h *grpcHandler) UpdateRecord(ctx context.Context, req *pb.UpdateRecordRequest) (*pb.Record, error) {
	data := req.GetData().AsMap()
	record, err := h.svc.UpdateRecord(ctx, req.GetEntity(), req.GetId(), data)
	if err != nil {
		return nil, err
	}
	return recordToProto(record), nil
}

func (h *grpcHandler) DeleteRecord(ctx context.Context, req *pb.DeleteRecordRequest) (*emptypb.Empty, error) {
	err := h.svc.DeleteRecord(ctx, req.GetEntity(), req.GetId())
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (h *grpcHandler) ListRecordHistory(ctx context.Context, req *pb.ListRecordHistoryRequest) (*pb.ListRecordHistoryResponse, error) {
	opts := HistoryOptions{
		PageSize:  int(req.GetPageSize()),
		PageToken: req.GetPageToken(),
	}

	history, nextToken, err := h.svc.RecordHistory(ctx, req.GetEntity(), req.GetId(), opts)
	if err != nil {
		return nil, err
	}

	pbHistory := make([]*pb.RecordHistoryEntry, 0, len(history))
	for _, h := range history {
		pbHistory = append(pbHistory, historyToProto(h))
	}

	return &pb.ListRecordHistoryResponse{
		History:       pbHistory,
		NextPageToken: nextToken,
	}, nil
}

func (h *grpcHandler) PivotRecords(ctx context.Context, req *pb.PivotRequest) (*pb.PivotResponse, error) {
	filters := make([]Filter, 0, len(req.GetFilters()))
	for _, f := range req.GetFilters() {
		filters = append(filters, Filter{
			FieldID:  f.GetField(),
			Operator: f.GetOperator(),
			Value:    f.GetValue().AsInterface(),
		})
	}

	opts := PivotOptions{
		RowField:    req.GetRowField(),
		ColumnField: req.GetColumnField(),
		Filters:     filters,
	}

	result, err := h.svc.PivotRecords(ctx, req.GetEntity(), opts)
	if err != nil {
		return nil, err
	}

	return pivotToProto(result), nil
}
