package researchidentity

import (
	"errors"
	"fmt"
	"sort"

	"github.com/david22573/ak-engine/internal/canonicalcontract"
)

func (d *Deriver) Derive(request DerivationRequest) (Assessment, error) {
	if d == nil {
		d = NewDeriver()
	}
	registry := d.registry
	if registry == nil {
		var err error
		registry, err = DefaultRegistry()
		if err != nil {
			return failAssessment(StatusCandidateIncomplete, "CANDIDATE_REGISTRY_INVALID", "candidate", err)
		}
	}
	candidate, err := registry.Resolve(request.RepositoryRoot, request.CandidateFamily, request.CandidateSide)
	if err != nil {
		return failAssessment(StatusCandidateIncomplete, "CANDIDATE_IDENTITY_DERIVATION_FAILED", "candidate", err)
	}
	configuration, err := ConfigurationHash(request.Configuration)
	if err != nil {
		return failAssessment(StatusConfigurationMissing, "CONFIGURATION_IDENTITY_DERIVATION_FAILED", "configuration", err)
	}
	engineSource, err := deriveEngineSource(d.sourceProvider, request.RepositoryRoot, request.Configuration.BuildTags)
	if err != nil {
		if derivation, ok := err.(*DerivationError); ok && derivation.Status == StatusDirtyEngineSource {
			return failAssessment(StatusDirtyEngineSource, derivation.Code, "engine_source", derivation.Err)
		}
		return failAssessment(StatusValidationFailed, "ENGINE_SOURCE_IDENTITY_DERIVATION_FAILED", "engine_source", err)
	}
	dataset, pit, err := verifyHistorianIdentity(request.HistorianManifestPath, request.DatasetRoot, request.ConsumedDatasetPaths, d.now().UTC())
	if err != nil {
		var derivation *DerivationError
		if errors.As(err, &derivation) {
			return failAssessment(derivation.Status, derivation.Code, "historian", derivation.Err)
		}
		return failAssessment(StatusDatasetIncomplete, "DATASET_IDENTITY_DERIVATION_FAILED", "historian", err)
	}
	if err := validateHistorianEvaluationScope(dataset, pit, request.Configuration); err != nil {
		return failAssessment(StatusConflict, "HISTORIAN_EVALUATION_SCOPE_CONFLICT", "cross_identity", err)
	}
	feature, err := deriveFeatureIdentity(request.RepositoryRoot, request.FeatureArtifactPath, request.FeatureRows, request.Configuration, engineSource, dataset)
	if err != nil {
		return failAssessment(StatusFeatureIncomplete, "FEATURE_IDENTITY_DERIVATION_FAILED", "feature", err)
	}
	var regimeIdentity *RegimeIdentity
	if candidate.UsesRegimes {
		derivedRegime, err := deriveRegimeIdentity(request.RepositoryRoot, request.RegimeArtifactPath, request.RegimeLabels, request.Configuration, engineSource, dataset, feature)
		if err != nil {
			return failAssessment(StatusRegimeIncomplete, "REGIME_IDENTITY_DERIVATION_FAILED", "regime", err)
		}
		regimeIdentity = &derivedRegime
	} else if request.RegimeArtifactPath != "" || len(request.RegimeLabels) != 0 {
		return failAssessment(StatusConflict, "UNDECLARED_REGIME_INPUT", "regime", fmt.Errorf("candidate declares no regimes but regime input was consumed"))
	}
	consumed, err := deriveConsumedInput(request, dataset, feature, regimeIdentity)
	if err != nil {
		return failAssessment(StatusConsumedIncomplete, "CONSUMED_INPUT_IDENTITY_DERIVATION_FAILED", "consumed_input", err)
	}
	series, err := deriveSeriesIdentity(request.Returns, request.Timestamps, request.Configuration.EvaluationStartMS, request.Configuration.EvaluationEndMS)
	if err != nil {
		return failAssessment(StatusSeriesIncomplete, "SERIES_IDENTITY_DERIVATION_FAILED", "series", err)
	}
	identity := BoundResearchIdentity{
		Contract:  canonicalcontract.NewHeader(boundIdentitySchemaName, canonicalContractVersion, boundIdentityRole),
		Candidate: candidate, Configuration: configuration, EngineSource: engineSource,
		Dataset: dataset, PIT: pit, Feature: feature, Regime: regimeIdentity,
		ConsumedInput:            consumed,
		Series:                   series,
		HistorianManifestHash:    dataset.ManifestHash,
		HistorianManifestRawHash: dataset.ManifestRawHash,
	}
	if err := validateCrossIdentity(identity, request); err != nil {
		return failAssessment(StatusConflict, "CROSS_IDENTITY_CONFLICT", "cross_identity", err)
	}
	identity.ArtifactHash, err = artifactHash(boundIdentitySchemaName, boundIdentityRole, identity)
	if err != nil {
		return failAssessment(StatusValidationFailed, "TOP_LEVEL_IDENTITY_HASH_FAILED", "top_level", err)
	}
	identity.IdentityHash = identity.ArtifactHash
	canonicalIdentity, err := canonicalcontract.CanonicalizeValue(identity)
	if err != nil {
		return failAssessment(StatusValidationFailed, "TOP_LEVEL_IDENTITY_CANONICALIZATION_FAILED", "top_level", err)
	}
	if _, err := canonicalcontract.ValidateArtifact(canonicalIdentity, true); err != nil {
		return failAssessment(StatusValidationFailed, "TOP_LEVEL_IDENTITY_CONTRACT_FAILED", "top_level", err)
	}
	return Assessment{Status: StatusComplete, Identity: &identity, Findings: []Finding{}}, nil
}

func validateHistorianEvaluationScope(dataset DatasetIdentity, pit PITEvidenceIdentity, config ResolvedResearchConfiguration) error {
	if !containsString(dataset.Symbols, config.Symbol) {
		return fmt.Errorf("Historian dataset universe does not contain configured symbol")
	}
	start, err := parseUTC(dataset.StartUTC)
	if err != nil {
		return err
	}
	end, err := parseUTC(dataset.EndUTC)
	if err != nil {
		return err
	}
	if start.UnixMilli() > config.EvaluationStartMS || end.UnixMilli() < config.EvaluationEndMS {
		return fmt.Errorf("Historian dataset window does not contain configured evaluation window")
	}
	if pit.EvaluationCutoffUTC != dataset.PointInTimeCutoffUTC {
		return fmt.Errorf("Historian PIT cutoff does not match dataset cutoff")
	}
	return nil
}

func failAssessment(status IdentityStatus, code, domain string, err error) (Assessment, error) {
	finding := Finding{Code: code, Domain: domain, Reason: err.Error(), Status: status, Blocking: true}
	derivation := &DerivationError{Status: status, Code: code, Err: err}
	return Assessment{Status: status, Findings: []Finding{finding}}, derivation
}

func validateCrossIdentity(identity BoundResearchIdentity, request DerivationRequest) error {
	if identity.Candidate.CandidateID == "" || identity.Candidate.CandidateVersion == "" || identity.Candidate.RegistrationRecordHash == "" || identity.Candidate.Implementation.ImplementationHash == "" {
		return fmt.Errorf("candidate identity is incomplete")
	}
	if identity.Configuration.Effective.Symbol != request.Configuration.Symbol || identity.Configuration.Hash == "" {
		return fmt.Errorf("configuration identity does not match evaluation")
	}
	if identity.EngineSource.Dirty || identity.EngineSource.CommitSHA == "" || identity.EngineSource.TreeSHA == "" {
		return fmt.Errorf("Engine source identity is incomplete")
	}
	if !containsString(identity.Dataset.Symbols, request.Configuration.Symbol) {
		return fmt.Errorf("dataset universe does not contain evaluated symbol")
	}
	if identity.PIT.Status != "PASS" || !identity.PIT.FullWindowCoverage || identity.PIT.EvaluationCutoffUTC != identity.Dataset.PointInTimeCutoffUTC || identity.PIT.PITPolicyID != identity.Dataset.AvailabilityPolicyID || identity.PIT.PITPolicyVersion != identity.Dataset.AvailabilityPolicyVersion || identity.PIT.PITPolicyHash != identity.Dataset.AvailabilityPolicyHash || identity.PIT.CoveragePolicyID != identity.Dataset.CoveragePolicyID || identity.PIT.CoveragePolicyVersion != identity.Dataset.CoveragePolicyVersion || identity.PIT.CoveragePolicyHash != identity.Dataset.CoveragePolicyHash || identity.PIT.SourceArchiveID != identity.Dataset.SourceArchiveID || identity.PIT.SourceArchiveHash != identity.Dataset.SourceArchiveHash {
		return fmt.Errorf("PIT identity does not match dataset/cutoff")
	}
	if identity.Feature.InputDatasetHash != identity.Dataset.DatasetHash || identity.Feature.WindowStartMS != request.Configuration.EvaluationStartMS || identity.Feature.WindowEndMS != request.Configuration.EvaluationEndMS || identity.Feature.ImplementationCommit != identity.EngineSource.CommitSHA || identity.Feature.RowCount != len(request.FeatureRows) {
		return fmt.Errorf("feature identity does not match dataset/configuration window")
	}
	if identity.Candidate.UsesRegimes {
		if identity.Regime == nil || identity.Regime.InputDatasetHash != identity.Dataset.DatasetHash || identity.Regime.InputFeatureHash != identity.Feature.OutputArtifactHash || identity.Regime.WindowStartMS != identity.Feature.WindowStartMS || identity.Regime.WindowEndMS != identity.Feature.WindowEndMS || identity.Regime.ImplementationCommit != identity.EngineSource.CommitSHA || identity.Regime.RowCount != len(request.RegimeLabels) {
			return fmt.Errorf("regime identity does not match candidate/dataset/feature window")
		}
	}
	if identity.ConsumedInput.DatasetHash != identity.Dataset.DatasetHash || identity.ConsumedInput.FeatureHash != identity.Feature.OutputArtifactHash {
		return fmt.Errorf("consumed-input identity does not match dataset/features")
	}
	if identity.Regime != nil && identity.ConsumedInput.RegimeHash != identity.Regime.OutputArtifactHash {
		return fmt.Errorf("consumed-input regime hash mismatch")
	}
	expectedInputSeries := 2
	if identity.Regime != nil {
		expectedInputSeries++
	}
	if identity.ConsumedInput.FeatureRowCount != len(request.FeatureRows) || identity.ConsumedInput.RegimeRowCount != len(request.RegimeLabels) || identity.ConsumedInput.CandleRowCount != len(request.Candles) || identity.ConsumedInput.EvaluationEventCount != len(request.EvaluationEventTimestamps) || identity.ConsumedInput.InputSeriesCount != expectedInputSeries {
		return fmt.Errorf("consumed-input counts do not match actual evaluation inputs")
	}
	if identity.Series.ObservationCount != len(request.Returns) || identity.Series.ReturnCount != len(request.Returns) || identity.Series.TimestampCount != len(request.Timestamps) {
		return fmt.Errorf("series counts do not match metric inputs")
	}
	if len(request.EvaluationEventTimestamps) != len(request.Timestamps) {
		return fmt.Errorf("evaluation event count does not match series count")
	}
	for i := range request.Timestamps {
		if request.Timestamps[i] != request.EvaluationEventTimestamps[i] {
			return fmt.Errorf("evaluation timestamp series differs at index %d", i)
		}
	}
	return nil
}

func containsString(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
}
