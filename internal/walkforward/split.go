package walkforward

import (
	"time"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

type Split struct {
	Index        int
	TrainStartMs int64
	TrainEndMs   int64
	TestStartMs  int64
	TestEndMs    int64
}

func GenerateSplits(fromMs, toMs int64, trainWindow, testWindow time.Duration) []Split {
	var splits []Split
	trainMs := int64(trainWindow.Milliseconds())
	testMs := int64(testWindow.Milliseconds())

	idx := 1
	for start := fromMs; start+trainMs < toMs; start += testMs {
		trainStart := start
		trainEnd := start + trainMs
		testStart := trainEnd
		testEnd := testStart + testMs

		if testEnd > toMs {
			break
		}

		splits = append(splits, Split{
			Index:        idx,
			TrainStartMs: trainStart,
			TrainEndMs:   trainEnd,
			TestStartMs:  testStart,
			TestEndMs:    testEnd,
		})
		idx++
	}
	return splits
}

func FilterCandles(candles []protocol.Candle, startMs, endMs int64) []protocol.Candle {
	var out []protocol.Candle
	for _, c := range candles {
		// inclusive start, exclusive end
		if c.CloseTimeMS >= startMs && c.CloseTimeMS < endMs {
			out = append(out, c)
		}
	}
	return out
}
