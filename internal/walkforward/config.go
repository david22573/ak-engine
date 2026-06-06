package walkforward

import "time"

type Config struct {
	TrainWindow     time.Duration
	TestWindow      time.Duration
	TopCandidates   int
	MinTrades       int
	MaxDrawdown     float64
	MaxLossStreak   int
	MinProfitFactor float64
	SweepProfile    string
	FixedParams     *Params
}
