BUF_RUN := env -u BUF_TOKEN buf

.PHONY: proto proto-go proto-web buf-deps tidy run web-install web-dev web-build demo-help

buf-deps:
	$(BUF_RUN) dep update

proto-go: buf-deps
	$(BUF_RUN) generate --template buf.gen.yaml

proto-web: buf-deps
	$(BUF_RUN) generate --template buf.gen.web.yaml

proto: proto-go proto-web

tidy:
	go mod tidy

run:
	go run ./cmd/server

web-install:
	cd web && npm install --registry=https://registry.npmjs.org/ --always-auth=false

web-dev:
	cd web && npm run dev -- --host

web-build:
	cd web && npm run build

demo-help:
	@echo "make proto       - generate Go + web protobuf code via Buf"
	@echo "make tidy        - resolve Go dependencies"
	@echo "make web-install - install frontend dependencies"
	@echo "make web-build   - build frontend assets"
	@echo "make run         - run the demo server"
