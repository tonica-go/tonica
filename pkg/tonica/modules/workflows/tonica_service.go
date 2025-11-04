package workflows

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/tonica-go/tonica/pkg/tonica/service"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/tonica-go/tonica/pkg/tonica/proto/workflows"
)

// NewTonicaService creates a new tonica service for workflows module.
// temporalClient must be initialized before calling this function.
func NewTonicaService(temporalClient client.Client) *service.Service {
	// Create closure that captures temporal client
	registerGRPCFunc := func(grpcServer *grpc.Server, svc *service.Service) {
		registerGRPCWithClient(grpcServer, svc, temporalClient)
	}

	return service.NewService(
		service.WithName("workflows"),
		service.WithGRPCAddr(":19003"),
		service.WithGRPC(registerGRPCFunc),
		service.WithGateway(registerGateway),
	)
}

// registerGRPCWithClient registers the workflows gRPC service with temporal client.
func registerGRPCWithClient(grpcServer *grpc.Server, svc *service.Service, temporalClient client.Client) {
	if temporalClient == nil {
		panic("temporal client not configured for workflows service")
	}

	// Create workflows service
	workflowSvc := NewService(temporalClient)

	// Register gRPC server
	handler := &grpcHandler{svc: workflowSvc}
	pb.RegisterWorkflowServiceServer(grpcServer, handler)
}

// registerGateway registers the workflows gateway.
func registerGateway(ctx context.Context, mux *runtime.ServeMux, target string, opts []grpc.DialOption) error {
	return pb.RegisterWorkflowServiceHandlerFromEndpoint(ctx, mux, target, opts)
}

// grpcHandler implements pb.WorkflowServiceServer
type grpcHandler struct {
	pb.UnimplementedWorkflowServiceServer
	svc *Service
}

func (h *grpcHandler) TriggerWorkflow(ctx context.Context, req *pb.TriggerWorkflowRequest) (*pb.TriggerWorkflowResponse, error) {
	// Default to waiting for completion (async=false) for backward compatibility
	waitForCompletion := true
	if req.Async != nil && req.GetAsync() {
		waitForCompletion = false
	}

	executionID, status, err := h.svc.Trigger(
		ctx,
		req.GetWorkflow(),
		req.GetEntity(),
		req.GetRecordId(),
		req.GetInput(),
		waitForCompletion,
	)
	if err != nil {
		return nil, err
	}

	return &pb.TriggerWorkflowResponse{
		ExecutionId: executionID,
		Status:      status,
	}, nil
}

func (h *grpcHandler) ListNamespaces(ctx context.Context, req *pb.ListNamespacesRequest) (*pb.ListNamespacesResponse, error) {
	res, err := h.svc.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	response := &pb.ListNamespacesResponse{}
	for _, namespace := range res {
		response.Namespaces = append(response.Namespaces, namespace)
	}
	return response, nil
}

func (h *grpcHandler) ListWorkflows(ctx context.Context, req *pb.ListWorkflowsRequest) (*pb.ListWorkflowsResponse, error) {
	res, _, err := h.svc.ListWorkflows(ctx, req.GetNamespace(), req.GetWorkflowType(), req.GetStatus(), req.GetPageSize(), req.GetPageToken(), req.GetSearchQuery())
	if err != nil {
		return nil, err
	}
	response := &pb.ListWorkflowsResponse{}
	for _, workflow := range res {
		response.Executions = append(response.Executions, workflow)
	}
	return response, nil
}

func (h *grpcHandler) GetWorkflow(ctx context.Context, req *pb.GetWorkflowRequest) (*pb.WorkflowDetails, error) {
	res, err := h.svc.GetWorkflow(ctx, req.GetNamespace(), req.GetWorkflowId(), req.GetRunId())
	if err != nil {
		return nil, err
	}
	response := &pb.WorkflowDetails{}
	response.Execution = res.Execution
	response.ExecutionTime = res.ExecutionTime
	response.Result = res.Result
	response.PendingActivities = append(response.PendingActivities, res.PendingActivities...)
	response.FailureMessage = res.FailureMessage
	response.Input = res.Input

	return response, nil
}

func (h *grpcHandler) GetWorkflowHistory(ctx context.Context, req *pb.GetWorkflowHistoryRequest) (*pb.GetWorkflowHistoryResponse, error) {
	res, _, err := h.svc.GetWorkflowHistory(ctx, req.GetNamespace(), req.GetWorkflowId(), req.GetRunId(), req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	response := &pb.GetWorkflowHistoryResponse{}
	response.History = append(response.History, res...)

	return response, nil
}

func (h *grpcHandler) TerminateWorkflow(ctx context.Context, req *pb.TerminateWorkflowRequest) (*emptypb.Empty, error) {
	err := h.svc.TerminateWorkflow(ctx, req.GetNamespace(), req.GetWorkflowId(), req.GetRunId(), req.GetReason())
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (h *grpcHandler) CancelWorkflow(ctx context.Context, req *pb.CancelWorkflowRequest) (*emptypb.Empty, error) {
	err := h.svc.CancelWorkflow(ctx, req.GetNamespace(), req.GetWorkflowId(), req.GetRunId())
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (h *grpcHandler) SignalWorkflow(ctx context.Context, req *pb.SignalWorkflowRequest) (*emptypb.Empty, error) {
	err := h.svc.SignalWorkflow(ctx, req.GetNamespace(), req.GetWorkflowId(), req.GetRunId(), req.GetSignalName(), req.GetInput())
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (h *grpcHandler) RestartWorkflow(ctx context.Context, req *pb.RestartWorkflowRequest) (*pb.RestartWorkflowResponse, error) {
	wfId, rID, err := h.svc.RestartWorkflow(ctx, req.GetNamespace(), req.GetWorkflowId(), req.GetRunId())
	if err != nil {
		return nil, err
	}
	return &pb.RestartWorkflowResponse{
		WorkflowId: wfId,
		RunId:      rID,
	}, nil
}

func (h *grpcHandler) ListSchedules(ctx context.Context, req *pb.ListSchedulesRequest) (*pb.ListSchedulesResponse, error) {
	res, _, err := h.svc.ListSchedules(ctx, req.GetNamespace(), req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	response := &pb.ListSchedulesResponse{}
	for _, schedule := range res {
		response.Schedules = append(response.Schedules, schedule)
	}
	return response, nil
}

func (h *grpcHandler) GetSchedule(ctx context.Context, req *pb.GetScheduleRequest) (*pb.Schedule, error) {
	res, err := h.svc.GetSchedule(ctx, req.GetNamespace(), req.GetScheduleId())
	if err != nil {
		return nil, err
	}

	response := &pb.Schedule{}
	response.ScheduleId = res.ScheduleId
	response.WorkflowType = res.WorkflowType
	response.CronExpression = res.CronExpression
	response.NextRunTime = res.NextRunTime
	response.LastRunTime = res.LastRunTime
	response.Paused = res.Paused
	response.Notes = res.Notes
	response.WorkflowInput = res.WorkflowInput

	return response, nil
}

func (h *grpcHandler) PauseSchedule(ctx context.Context, req *pb.PauseScheduleRequest) (*emptypb.Empty, error) {
	err := h.svc.PauseSchedule(ctx, req.GetNamespace(), req.GetScheduleId(), req.GetNote())
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (h *grpcHandler) UnpauseSchedule(ctx context.Context, req *pb.UnpauseScheduleRequest) (*emptypb.Empty, error) {
	err := h.svc.UnpauseSchedule(ctx, req.GetNamespace(), req.GetScheduleId(), req.GetNote())
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (h *grpcHandler) TriggerSchedule(ctx context.Context, req *pb.TriggerScheduleRequest) (*pb.TriggerWorkflowResponse, error) {
	id, status, err := h.svc.TriggerSchedule(ctx, req.GetNamespace(), req.GetScheduleId())
	if err != nil {
		return nil, err
	}
	return &pb.TriggerWorkflowResponse{
		ExecutionId: id,
		Status:      status,
	}, nil
}
