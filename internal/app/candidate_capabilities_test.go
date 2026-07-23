package app

import (
	"testing"
)

func TestCandidateCapabilities(t *testing.T) {
	t.Run("context-free candidate accepts missing context", func(t *testing.T) {
		caps := CandidateCapabilities{
			RequiresBTCContext:            false,
			RequiresETHContext:            false,
			ContextFreeModeAllowed:        true,
			AllowedMissingContextBehavior: AllowMissingContext,
			IsResearchOnly:                true,
			IsPromotable:                  false,
		}
		err := ValidateCandidateInputs(caps, false, false, false, false)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("context-dependent candidate rejects missing BTC context", func(t *testing.T) {
		caps := CandidateCapabilities{
			RequiresBTCContext:            true,
			RequiresETHContext:            true,
			ContextFreeModeAllowed:        false,
			AllowedMissingContextBehavior: RejectMissingContext,
			IsResearchOnly:                true,
			IsPromotable:                  false,
		}
		err := ValidateCandidateInputs(caps, false, true, false, false)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("context-dependent candidate rejects missing ETH context", func(t *testing.T) {
		caps := CandidateCapabilities{
			RequiresBTCContext:            true,
			RequiresETHContext:            true,
			ContextFreeModeAllowed:        false,
			AllowedMissingContextBehavior: RejectMissingContext,
			IsResearchOnly:                true,
			IsPromotable:                  false,
		}
		err := ValidateCandidateInputs(caps, true, false, false, false)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("no candidate is promotable", func(t *testing.T) {
		caps := CandidateCapabilities{
			IsPromotable: true,
		}
		err := ValidateCandidateInputs(caps, true, true, true, false)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("candidate that does not support compact emission cannot emit compact events", func(t *testing.T) {
		caps := CandidateCapabilities{
			SupportsCompactEmission: false,
			IsResearchOnly:          true,
			IsPromotable:            false,
			ContextFreeModeAllowed:  true,
		}
		err := ValidateCandidateInputs(caps, true, true, true, true)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("compact-emission-supported candidate can emit compact events", func(t *testing.T) {
		caps := CandidateCapabilities{
			SupportsCompactEmission: true,
			IsResearchOnly:          true,
			IsPromotable:            false,
			ContextFreeModeAllowed:  true,
		}
		err := ValidateCandidateInputs(caps, true, true, true, true)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}
