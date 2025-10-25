package payment

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	paymentv1 "github.com/tonica-go/tonica/example/dev/proto/payment/v1"
	"github.com/tonica-go/tonica/pkg/tonica/service"
	"google.golang.org/grpc"
)

type PaymentsServiceServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	srv *service.Service
}

func RegisterGRPC(s *grpc.Server, srv *service.Service) {
	paymentv1.RegisterPaymentServiceServer(s, &PaymentsServiceServer{srv: srv})
}

func RegisterGateway(ctx context.Context, mux *runtime.ServeMux, target string, dialOpts []grpc.DialOption) error {
	return paymentv1.RegisterPaymentServiceHandlerFromEndpoint(ctx, mux, target, dialOpts)
}

func GetClient(s *grpc.ClientConn) paymentv1.PaymentServiceClient {
	return paymentv1.NewPaymentServiceClient(s)
}

func (s *PaymentsServiceServer) Auth(ctx context.Context, in *paymentv1.AuthRequest) (*paymentv1.AuthResponse, error) {
	return &paymentv1.AuthResponse{}, nil
}
func (s *PaymentsServiceServer) Profile(ctx context.Context, in *paymentv1.ProfileRequest) (*paymentv1.ProfileResponse, error) {
	return &paymentv1.ProfileResponse{}, nil
}
func (s *PaymentsServiceServer) Webhook(ctx context.Context, in *paymentv1.WebhookRequest) (*paymentv1.WebhookResponse, error) {
	return &paymentv1.WebhookResponse{}, nil
}
