package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	demov1 "github.com/minauteur/connect-demo/internal/gen/demo/v1"
	demov1connect "github.com/minauteur/connect-demo/internal/gen/demo/v1/demov1connect"
	"github.com/minauteur/connect-demo/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	grpcAddress := "127.0.0.1:8081"
	httpAddress := ":8080"

	grpcServer := grpc.NewServer()
	grpcSvc := &service.GRPCDemoService{}
	demov1.RegisterDemoServiceServer(grpcServer, grpcSvc)

	grpcListener, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		log.Fatalf("listen gRPC: %v", err)
	}

	go func() {
		log.Printf("gRPC (plain protobuf) listening on %s", grpcAddress)
		if serveErr := grpcServer.Serve(grpcListener); serveErr != nil {
			log.Printf("gRPC server stopped: %v", serveErr)
		}
	}()

	gatewayMux := runtime.NewServeMux()
	err = demov1.RegisterDemoServiceHandlerFromEndpoint(
		ctx,
		gatewayMux,
		grpcAddress,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	)
	if err != nil {
		log.Fatalf("register grpc-gateway: %v", err)
	}

	connectPath, connectHandler := demov1connect.NewDemoServiceHandler(&service.ConnectDemoService{})

	rootMux := http.NewServeMux()
	rootMux.Handle(connectPath, connectHandler)
	rootMux.Handle("/v1/", gatewayMux)
	rootMux.Handle("/", http.FileServer(http.Dir("web/dist")))

	httpServer := &http.Server{
		Addr:    httpAddress,
		Handler: withCORS(rootMux),
	}

	go func() {
		log.Printf("HTTP listening on %s", httpAddress)
		log.Printf("Open http://localhost:8080 for the browser demo")
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.Printf("HTTP server stopped: %v", serveErr)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
	grpcServer.GracefulStop()
	log.Println("Shutdown complete")
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
