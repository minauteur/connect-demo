# Connect Demo: protobuf vs grpc-gateway vs Connect vs grpc-web

This project is a minimal interactive demo showing four protobuf-based API styles:

1. plain protobuf over gRPC (`grpc-go`) on `127.0.0.1:8080`
2. `grpc-gateway` JSON transcoding on `http://localhost:8080/v1/hello:echo`
3. Buf Connect protocol on `http://localhost:8080/demo.v1.DemoService/SayHello`
4. grpc-web protocol on `http://localhost:8080/demo.v1.DemoService/SayHello`

The browser UI focuses on Buf Connect, but you can switch to grpc-web and grpc-gateway for comparison.

## Quick start

```bash
make proto
make tidy
make web-install
make web-build
make run
```

Then open `http://localhost:8080`.

## Plain gRPC check (terminal)

```bash
grpcurl -plaintext -d '{"name":"Mitch"}' 127.0.0.1:8080 demo.v1.DemoService/SayHello
```

## Why this layout

- Go protobuf output is generated into `internal/gen` (keeps generated artifacts private to the module).
- Browser protobuf types are generated into `web/src/gen`.
- A single proto definition drives all transports.
