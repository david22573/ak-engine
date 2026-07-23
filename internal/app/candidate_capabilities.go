package app

import (
	"errors"
)

type MissingContextBehavior string

const (
	RejectMissingContext MissingContextBehavior = "reject"
	AllowMissingContext  MissingContextBehavior = "allow"
)

type CandidateCapabilities struct {
	CandidateName                 string
	FamilyName                    string
	SupportedSymbols              []string
	SupportedIntervals            []string
	SupportedHorizons             []string
	RequiresBTCContext            bool
	RequiresETHContext            bool
	RequiresFundingContext        bool
	RequiresVolumeContext         bool
	RequiresClusterContext        bool
	SupportsCompactEmission       bool
	ContextFreeModeAllowed        bool
	AllowedMissingContextBehavior MissingContextBehavior
	IsResearchOnly                bool
	IsPromotable                  bool
}

func ValidateCandidateInputs(caps CandidateCapabilities, hasBTC, hasETH, hasFunding bool, emitCompact bool) error {
	if caps.IsPromotable {
		return errors.New("no candidate in this phase is promotable")
	}

	if !caps.IsResearchOnly {
		return errors.New("candidate must be research-only in this phase")
	}

	if caps.RequiresBTCContext && !hasBTC {
		if caps.AllowedMissingContextBehavior == RejectMissingContext || !caps.ContextFreeModeAllowed {
			return errors.New("missing BTC context: context-dependent candidate requires it")
		}
	}

	if caps.RequiresETHContext && !hasETH {
		if caps.AllowedMissingContextBehavior == RejectMissingContext || !caps.ContextFreeModeAllowed {
			return errors.New("missing ETH context: context-dependent candidate requires it")
		}
	}

	if !caps.ContextFreeModeAllowed && (!hasBTC || !hasETH) {
		return errors.New("missing context and context-free mode is not allowed")
	}

	if emitCompact && !caps.SupportsCompactEmission {
		return errors.New("compact event emission requested but candidate does not support it")
	}

	return nil
}
