package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	demov1 "github.com/minauteur/connect-demo/internal/gen/demo/v1"
	demov1connect "github.com/minauteur/connect-demo/internal/gen/demo/v1/demov1connect"
	"github.com/minauteur/connect-demo/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slogger := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey, slog.LevelKey:
				if len(groups) == 0 {
					return slog.Attr{}
				}
			}
			return a
		}})
	slog.SetDefault(slog.New(slogger))

	grpcAddress := "127.0.0.1:8080"
	httpAddress := ":8080"

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(grpcServerInterceptor))
	grpcSvc := &service.GRPCDemoService{}
	demov1.RegisterDemoServiceServer(grpcServer, grpcSvc)
	reflection.Register(grpcServer)

	gatewayMux := runtime.NewServeMux()
	err := demov1.RegisterDemoServiceHandlerFromEndpoint(
		ctx,
		gatewayMux,
		grpcAddress,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithUnaryInterceptor(newGatewayInterceptor)},
	)
	if err != nil {
		slog.Error("register grpc-gateway", "error", err)
		panic(err)
	}
	connectInterceptors := connect.WithInterceptors(newConnectInterceptor())
	connectPath, connectHandler := demov1connect.NewDemoServiceHandler(&service.ConnectDemoService{}, connectInterceptors)

	rootMux := http.NewServeMux()
	rootMux.Handle(connectPath, connectHandler)
	rootMux.Handle("/v1/", gatewayMux)
	rootMux.Handle("/", http.FileServer(http.Dir("web/dist")))

	webHandler := httpLoggingMiddleware(withCORS(rootMux))
	multiplexedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
			return
		}
		webHandler.ServeHTTP(w, r)
	})

	httpServer := &http.Server{
		Addr:    httpAddress,
		Handler: multiplexedHandler,
	}

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	httpServer.Protocols = protocols

	go func() {
		slog.Info("Multiplexed server (gRPC + Connect + grpc-web + gateway) listening", "address", httpAddress)
		slog.Info("Open http://localhost:8080 for the browser demo")
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			slog.Error("HTTP server stopped", "error", serveErr)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP shutdown error", "error", err)
	}
	grpcServer.GracefulStop()
	slog.Info("Shutdown complete")
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Connect-Protocol-Version,Connect-Timeout-Ms,Grpc-Timeout,X-Grpc-Web,X-User-Agent")
		w.Header().Set("Access-Control-Expose-Headers", "Grpc-Status,Grpc-Message,Grpc-Status-Details-Bin,Demo-Protocol")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Client side interceptor for intercepting requests made by the grpc-gateway client to the gRPC server
func newGatewayInterceptor(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	slog.Info("GATEWAY CLIENT: Incoming request for...", "protocol", "grpc-gateway", "method", method)
	return invoker(ctx, method, req, reply, cc, opts...)
}

// Connect server-side interceptor for intercepting requests prior to handling by the Connect server
func newConnectInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			slog.Info("CONNECT SERVER: Incoming grpc request for...", "protocol", req.Peer().Protocol, "method", req.Spec().Procedure)
			return next(ctx, req)
		}
	}
}

// HTTP logging middleware logs the HTTP request type (method and path) before any server-side processing occurs.
func httpLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the HTTP request type (method and path) before gRPC-Gateway processes it
		slog.Info("HTTP: Incoming http request for...", "method", r.Method, "path", r.URL.Path, "content-type", r.Header.Get("Content-Type"))

		// Pass to the gRPC-Gateway mux
		next.ServeHTTP(w, r)
	})
}

// Server-side interceptor intercepts requests after any transposition, but before the gRPC server has handled them
func grpcServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Pre-processing
	slog.Info("GRPC SERVER: Incoming grpc server request for...", "method", info.FullMethod)

	// Call the actual RPC handler
	resp, err := handler(ctx, req)

	return resp, err
}
