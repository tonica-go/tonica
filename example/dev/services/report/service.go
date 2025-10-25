package report

import (
	"context"

	reportsv1 "github.com/alexrett/tonica/example/dev/proto/reports/v1"
	"github.com/alexrett/tonica/pkg/tonica/service"
	"google.golang.org/grpc"
)

type ReportsServiceServer struct {
	reportsv1.UnimplementedReportsServiceServer
	srv *service.Service
}

func RegisterGRPC(s *grpc.Server, srv *service.Service) {
	reportsv1.RegisterReportsServiceServer(s, &ReportsServiceServer{srv: srv})
}

func GetClient(s *grpc.ClientConn) reportsv1.ReportsServiceClient {
	return reportsv1.NewReportsServiceClient(s)
}

func (s *ReportsServiceServer) ListReports(ctx context.Context, in *reportsv1.ListReportsRequest) (*reportsv1.ListReportsResponse, error) {
	return &reportsv1.ListReportsResponse{}, nil
}
func (s *ReportsServiceServer) GetReport(ctx context.Context, in *reportsv1.GetReportRequest) (*reportsv1.GetReportResponse, error) {
	return &reportsv1.GetReportResponse{
		Report: &reportsv1.ReportItem{
			Id:        "1",
			Title:     "2",
			CreatedAt: "3",
		},
	}, nil
}
