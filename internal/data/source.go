package data

import (
	"context"
	"fmt"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

type CandleSource interface {
	LoadCandles(ctx context.Context, req CandleRequest) ([]protocol.Candle, error)
	Name() string
}

func NewCandleSource(source string) (CandleSource, error) {
	switch source {
	case "local-json":
		return NewLocalJSONSource(), nil
	case "local-parquet":
		return NewLocalParquetSource(), nil
	case "r2":
		return NewR2ParquetSource(), nil
	default:
		return nil, fmt.Errorf("unknown source %q; supported sources are: local-json, local-parquet, r2", source)
	}
}
