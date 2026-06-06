package strategy

import (
	"context"

	"github.com/davidmiguel22573/ak-engine/pkg/protocol"
)

type Side string

const (
	SideNone  Side = ""
	SideLong  Side = "long"
	SideShort Side = "short"
)

type EntryVariant string

const (
	EntryVariantScore                EntryVariant = "score"
	EntryVariantPullbackReclaim      EntryVariant = "pullback_reclaim"
	EntryVariantBreakoutRetest       EntryVariant = "breakout_retest"
	EntryVariantMomentumContinuation EntryVariant = "momentum_continuation"
)

type ExitModel string

const (
	ExitModelFixedTPSL        ExitModel = "fixed_tp_sl"
	ExitModelTimeStop         ExitModel = "time_stop"
	ExitModelPartialTPTrail   ExitModel = "partial_tp_trail"
	ExitModelBreakevenAfter1R ExitModel = "breakeven_after_1r"
	ExitModelTrailAfterMFE    ExitModel = "trail_after_mfe"
	ExitModelCutIfNoProgress  ExitModel = "cut_if_no_progress"
)

type ExitPlan struct {
	Model                     ExitModel `json:"model"`
	PartialTakeProfitR        float64   `json:"partial_take_profit_r,omitempty"`
	PartialTakeProfitFraction float64   `json:"partial_take_profit_fraction,omitempty"`
	BreakevenTriggerR         float64   `json:"breakeven_trigger_r,omitempty"`
	TrailAfterMFER            float64   `json:"trail_after_mfe_r,omitempty"`
	TrailDistanceR            float64   `json:"trail_distance_r,omitempty"`
	CutNoProgressR            float64   `json:"cut_no_progress_r,omitempty"`
	CutNoProgressCandles      int       `json:"cut_no_progress_candles,omitempty"`
}

type Signal struct {
	Side           Side
	StopLossBPS    float64
	TakeProfitBPS  float64
	MaxHoldCandles int
	RiskFraction   float64
	ClosePosition  bool
	Decision       *WindowDecision
	ExitPlan       ExitPlan
}

type State struct {
	Candles            []protocol.Candle
	HasPosition        bool
	PositionSide       Side
	PositionEntryPrice float64
	HeldCandles        int
}

type Strategy interface {
	Name() string
	OnCandle(ctx context.Context, state State, candle protocol.Candle) (Signal, error)
}
