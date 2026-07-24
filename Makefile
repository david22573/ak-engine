.PHONY: fmt test vet build ci verify run-version proof-local proof-backtest-local proof-fast-accumulation-local proof-fast-accumulation-diagnostics-local proof-fast-accumulation-sweep-local proof-walk-forward-local proof-fast-accumulation-strict-local proof-walk-forward-strict-local proof-fast-accumulation-calibration-local proof-walk-forward-calibration-local proof-fast-accumulation-economics-local proof-fast-accumulation-entry-variants-local proof-r2 termux-worker-remote-mkdir termux-worker-push termux-worker-pull-summaries termux-worker-pull-reports phone-raw-count phone-backup-raw-to-drive phase10-9c-coverage phase10-9c-run-symbol phase10-9c-raw-files help
 
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

## verify: Run deterministic standalone verification with GOWORK=off
verify:
	./scripts/verify.sh

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

## termux-worker-remote-mkdir: Create the remote repo directory using termux_worker.env or PHONE_HOST
termux-worker-remote-mkdir:
	./scripts/termux_worker_sync.sh remote-mkdir

## termux-worker-push: Update the phone git checkout, then push synced repo files
termux-worker-push:
	./scripts/termux_worker_sync.sh push

## termux-worker-pull-summaries: Pull compact summaries from the Termux phone worker
termux-worker-pull-summaries:
	./scripts/termux_worker_sync.sh pull-summaries

## termux-worker-pull-reports: Pull phase reports and chunk outputs from the Termux phone worker
termux-worker-pull-reports:
	./scripts/termux_worker_sync.sh pull-reports

## phone-raw-count: Count raw funding-event gzip files under runs/reports/chunks
phone-raw-count:
	./scripts/backup_raw_to_drive.sh count

## phone-backup-raw-to-drive: Copy and verify phone raw funding-event gzip files with rclone
phone-backup-raw-to-drive:
	./scripts/backup_raw_to_drive.sh backup

## phase10-9c-coverage: Scan retained compact-summary coverage and ranked inventory
phase10-9c-coverage:
	./scripts/phase10_9c_phone_worker.sh coverage

## phase10-9c-run-symbol: Run one-symbol 10.9C funding regeneration with post-aggregate raw cleanup
phase10-9c-run-symbol:
	@if [ -z "$(SYMBOL)" ]; then echo "SYMBOL is required"; exit 1; fi
	./scripts/phase10_9c_phone_worker.sh run-symbol "$(SYMBOL)" "$(if $(FROM),$(FROM),2024-01)" "$(if $(TO),$(TO),2025-12)"

## phase10-9c-raw-files: Show remaining heavy raw files under runs/
phase10-9c-raw-files:
	./scripts/phase10_9c_phone_worker.sh raw-files "$(SYMBOL)"

## help: Show help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
