.PHONY: fmt test vet build ci run-version proof-local proof-r2 help
 
# Default target
all: build

## fmt: Run go fmt
fmt:
	go fmt ./...

## test: Run tests
test:
	go test -v ./...

## vet: Run go vet
vet:
	go vet ./...

## build: Build the binary
build:
	go build -o ./bin/ak-engine ./cmd/ak-engine

## ci: Run all CI checks
ci: fmt vet test build

## run-version: Run the version command
run-version:
	go run ./cmd/ak-engine version

## proof-local: Run local proof flow
proof-local: ci
	go run ./cmd/ak-engine inspect-dataset --source local-json --path testdata/candles/btc_5m_sample.json --market futures-um --symbol BTCUSDT --interval 5m --from 2024-01-01 --to 2024-01-02 --format json

## proof-r2: Run R2 proof flow
proof-r2: ci
	@if [ -z "$$R2_ACCOUNT_ID" ] || [ -z "$$R2_ACCESS_KEY_ID" ] || [ -z "$$R2_SECRET_ACCESS_KEY" ] || [ -z "$$R2_BUCKET_NAME" ]; then \
		echo "Error: Missing R2 environment variables. Requires: R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, R2_BUCKET_NAME"; \
		exit 1; \
	fi
	go run ./cmd/ak-engine inspect-dataset --source r2 --market futures-um --symbol LINKUSDT --interval 1m --from 2023-01-01 --to 2023-01-31 --format json

## help: Show help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
