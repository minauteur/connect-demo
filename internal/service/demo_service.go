package service

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	demov1 "github.com/minauteur/connect-demo/internal/gen/demo/v1"
)

// GRPCDemoService powers the plain gRPC endpoint.
type GRPCDemoService struct {
	demov1.UnimplementedDemoServiceServer
}

func (s *GRPCDemoService) SayHello(_ context.Context, req *demov1.HelloRequest) (*demov1.HelloResponse, error) {
	return buildHelloResponse(req.GetName(), "plain protobuf over gRPC"), nil
}

// ConnectDemoService powers the Connect endpoint and can report the wire protocol in use.
type ConnectDemoService struct{}

func (s *ConnectDemoService) SayHello(_ context.Context, req *connect.Request[demov1.HelloRequest]) (*connect.Response[demov1.HelloResponse], error) {
	protocol := "buf connect"
	if strings.EqualFold(req.Peer().Protocol, "grpcweb") {
		protocol = "grpc-web"
	}

	res := connect.NewResponse(buildHelloResponse(req.Msg.GetName(), protocol))
	res.Header().Set("Demo-Protocol", protocol)
	return res, nil
}

func buildHelloResponse(name string, protocol string) *demov1.HelloResponse {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		trimmedName = "friend"
	}

	return &demov1.HelloResponse{
		Message:      fmt.Sprintf("Hello, %s!", trimmedName),
		ProtocolNote: fmt.Sprintf("Request served using %s.", protocol),
	}
}
