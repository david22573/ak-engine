.PHONY: fmt test vet build ci run-version proof-local proof-backtest-local proof-fast-accumulation-local proof-fast-accumulation-diagnostics-local proof-fast-accumulation-sweep-local proof-walk-forward-local proof-fast-accumulation-strict-local proof-walk-forward-strict-local proof-fast-accumulation-calibration-local proof-walk-forward-calibration-local proof-fast-accumulation-economics-local proof-fast-accumulation-entry-variants-local proof-r2 help
 
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

## proof-backtest-local: Run local backtest proof flow
proof-backtest-local: ci
	go run ./cmd/ak-engine backtest \
		--source local-json \
		--path testdata/candles/btc_5m_sample.json \
		--market futures-um \
		--symbol BTCUSDT \
		--interval 5m \
		--from 2024-01-01 \
		--to 2024-01-02 \
		--strategy baseline \
		--format json

## proof-fast-accumulation-local: Run local Fast Accumulation proof flow
proof-fast-accumulation-local: ci
	go run ./cmd/ak-engine backtest \
		--source local-json \
		--path testdata/candles/btc_5m_fast_accumulation_sample.json \
		--market futures-um \
		--symbol BTCUSDT \
		--interval 5m \
		--from 2024-01-01 \
		--to 2024-01-02 \
		--strategy fast_accumulation \
		--format json

## proof-fast-accumulation-diagnostics-local: Run local Fast Accumulation diagnostics proof flow
proof-fast-accumulation-diagnostics-local: ci
	go run ./cmd/ak-engine backtest \
		--source local-json \
		--path testdata/candles/btc_5m_fast_accumulation_sample.json \
		--market futures-um \
		--symbol BTCUSDT \
		--interval 5m \
		--from 2024-01-01 \
		--to 2024-01-02 \
		--strategy fast_accumulation \
		--format json \
		--include-decisions

## proof-fast-accumulation-sweep-local: Run local Fast Accumulation parameter sweep proof flow
proof-fast-accumulation-sweep-local: ci
	go run ./cmd/ak-engine sweep \
		--source local-json \
		--path testdata/candles/btc_5m_fast_accumulation_sample.json \
		--market futures-um \
		--symbol BTCUSDT \
		--interval 5m \
		--from 2024-01-01 \
		--to 2024-01-02 \
		--strategy fast_accumulation \
		--format json

## proof-walk-forward-local: Run local Fast Accumulation walk-forward proof flow
proof-walk-forward-local: ci
	go run ./cmd/ak-engine walk-forward \
		--source local-json \
		--path testdata/candles/btc_5m_walk_forward_sample.json \
		--market futures-um \
		--symbol BTCUSDT \
		--interval 5m \
		--from 2024-01-01 \
		--to 2024-01-02 \
		--strategy fast_accumulation \
		--train-window 45m \
		--test-window 15m \
		--format json

## proof-fast-accumulation-strict-local: Run local strict Fast Accumulation proof flow
proof-fast-accumulation-strict-local: ci
	go run ./cmd/ak-engine backtest \
		--source local-json \
		--path testdata/candles/btc_5m_fast_accumulation_sample.json \
		--market futures-um \
		--symbol BTCUSDT \
		--interval 5m \
		--from 2024-01-01 \
		--to 2024-01-02 \
		--strategy fast_accumulation_strict \
		--format json

## proof-walk-forward-strict-local: Run local strict Fast Accumulation walk-forward proof flow
proof-walk-forward-strict-local: ci
	go run ./cmd/ak-engine walk-forward \
		--source local-json \
		--path testdata/candles/btc_5m_walk_forward_sample.json \
		--market futures-um \
		--symbol BTCUSDT \
		--interval 5m \
		--from 2024-01-01 \
		--to 2024-01-02 \
		--strategy fast_accumulation_strict \
		--train-window 45m \
		--test-window 15m \
		--format json

## proof-fast-accumulation-calibration-local: Run local calibration-preset Fast Accumulation proof flow
proof-fast-accumulation-calibration-local: ci
	go run ./cmd/ak-engine backtest \
		--source local-json \
		--path testdata/candles/btc_5m_fast_accumulation_sample.json \
		--market futures-um \
		--symbol BTCUSDT \
		--interval 5m \
		--from 2024-01-01 \
		--to 2024-01-02 \
		--strategy fast_accumulation_strict_no_70_84_longs \
		--format json

## proof-walk-forward-calibration-local: Run local calibration-preset walk-forward proof flow
proof-walk-forward-calibration-local: ci
	go run ./cmd/ak-engine walk-forward \
		--source local-json \
		--path testdata/candles/btc_5m_walk_forward_sample.json \
		--market futures-um \
		--symbol BTCUSDT \
		--interval 5m \
		--from 2024-01-01 \
		--to 2024-01-02 \
		--strategy fast_accumulation_strict_low_frequency \
		--train-window 45m \
		--test-window 15m \
		--min-trades 0 \
		--format json

## proof-fast-accumulation-economics-local: Run local economics diagnostics proof flow
proof-fast-accumulation-economics-local: ci
	go run ./cmd/ak-engine backtest \
		--source local-json \
		--path testdata/candles/btc_5m_fast_accumulation_sample.json \
		--market futures-um \
		--symbol BTCUSDT \
		--interval 5m \
		--from 2024-01-01 \
		--to 2024-01-02 \
		--strategy fast_accumulation_economics_guard \
		--format json

## proof-fast-accumulation-entry-variants-local: Run local entry-variant proof flow
proof-fast-accumulation-entry-variants-local: ci
	go run ./cmd/ak-engine backtest \
		--source local-json \
		--path testdata/candles/btc_5m_fast_accumulation_sample.json \
		--market futures-um \
		--symbol BTCUSDT \
		--interval 5m \
		--from 2024-01-01 \
		--to 2024-01-02 \
		--strategy fast_accumulation_pullback_reclaim \
		--format json

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
