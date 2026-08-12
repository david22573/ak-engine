package rifbridge

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/david22573/ak-engine/internal/canonicalcontract"
	"github.com/david22573/ak-engine/internal/executionseries"
	"github.com/david22573/ak-engine/internal/researchidentity"
)

// IdentityDeriver is the narrow identity boundary owned by the bridge. The
// production bridge always uses researchidentity.NewDeriver; injection exists
// only to make repository-state fixtures deterministic in tests and smoke.
type IdentityDeriver interface {
	Derive(researchidentity.DerivationRequest) (researchidentity.Assessment, error)
}

// Bridge emits Engine-local research diagnostics. It has no RIF lifecycle role.
type Bridge struct {
	identityDeriver IdentityDeriver
}

func NewBridge() *Bridge {
	return &Bridge{identityDeriver: researchidentity.NewDeriver()}
}

func NewBridgeWithDeriver(deriver IdentityDeriver) *Bridge {
	if deriver == nil {
		deriver = researchidentity.NewDeriver()
	}
	return &Bridge{identityDeriver: deriver}
}

// EmitResearchDiagnostics derives exact identity, evaluates local integrity,
// and emits one explicitly non-authoritative diagnostic artifact. Derivation
// failures remain visible in the local artifact and are returned to the caller.
func (b *Bridge) EmitResearchDiagnostics(input ResearchAssessment) (ResearchDiagnosticsResult, error) {
	result := ResearchDiagnosticsResult{
		ArtifactDisposition: ArtifactSuppressed,
		Failure:             DiagnosticsFailureInvalidInput,
	}
	if err := validateResearchAssessment(input); err != nil {
		return result, err
	}
	if b == nil || b.identityDeriver == nil {
		b = NewBridge()
	}

	identityAssessment, identityErr := b.identityDeriver.Derive(input.IdentityRequest)
	identityAssessment, identityErr = normalizeIdentityAssessment(identityAssessment, identityErr)
	if identityAssessment.Status == researchidentity.StatusComplete && (identityAssessment.Identity == nil || identityAssessment.Identity.Series.SeriesGenerationVersion != input.ExecutionSeriesGeneration) {
		return result, fmt.Errorf("%w: classification series does not match derived evaluation series", ErrInvalidResearchInput)
	}
	if seriesErr := validateBridgeMetricSeries(input.IdentityRequest.Returns, input.IdentityRequest.Timestamps); seriesErr != nil && identityAssessment.Status == researchidentity.StatusComplete {
		identityAssessment = researchidentity.Assessment{Status: researchidentity.StatusSeriesIncomplete, Findings: []researchidentity.Finding{{
			Code: "BRIDGE_SERIES_VALIDATION_FAILED", Domain: "series", Reason: seriesErr.Error(), Status: researchidentity.StatusSeriesIncomplete, Blocking: true,
		}}}
		identityErr = &researchidentity.DerivationError{Status: researchidentity.StatusSeriesIncomplete, Code: "BRIDGE_SERIES_VALIDATION_FAILED", Err: seriesErr}
	}

	identityContractFindings := identityFindings(identityAssessment.Findings)
	localFindings := make([]researchidentity.Finding, 0, 4)
	returns := input.IdentityRequest.Returns
	timestamps := input.IdentityRequest.Timestamps
	config := input.IdentityRequest.Configuration

	minimumSamples := config.DiagnosticMinimumSamples
	if minimumSamples <= 0 {
		minimumSamples = 30
	}
	if err := checkSampleSize(len(timestamps), minimumSamples); err != nil {
		localFindings = append(localFindings, newResearchFinding(
			"RESEARCH_LOW_SAMPLE_SIZE", "ERROR", err.Error(), true,
			"Increase the evaluation window or use an explicitly reviewed sample-size configuration.",
		))
	}
	if err := checkLookAheadBias(timestamps); err != nil {
		localFindings = append(localFindings, newResearchFinding(
			"RESEARCH_TIMESTAMP_ORDER_INVALID", "ERROR", err.Error(), true,
			"Provide a strictly increasing timestamp series aligned one-for-one with returns.",
		))
	}
	observationsPerParameter := config.ObservationsPerParameter
	if observationsPerParameter <= 0 || math.IsNaN(observationsPerParameter) || math.IsInf(observationsPerParameter, 0) {
		observationsPerParameter = 20
	}
	if err := checkParameterMining(len(timestamps), config.ModelParameterCount, observationsPerParameter); err != nil {
		localFindings = append(localFindings, newResearchFinding(
			"RESEARCH_PARAMETER_MINING_RISK", "ERROR", err.Error(), true,
			"Reduce the parameter count or increase the sample size.",
		))
	}

	metrics, metricsErr := calculateMetrics(returns, config.MetricRiskFreeRate, config.MetricPeriodsPerYear)
	if metricsErr != nil {
		localFindings = append(localFindings, newResearchFinding(
			"RESEARCH_METRICS_UNAVAILABLE", "ERROR", metricsErr.Error(), true,
			"Provide a finite, non-empty return series and valid metric configuration.",
		))
	}

	localIntegrity := LocalIntegrityPassed
	blockingFindings := make([]researchidentity.Finding, 0, len(localFindings))
	nonBlockingWarnings := make([]researchidentity.Finding, 0, len(localFindings))
	for _, finding := range localFindings {
		if finding.Blocking {
			localIntegrity = LocalIntegrityFailed
			blockingFindings = append(blockingFindings, finding)
		} else {
			nonBlockingWarnings = append(nonBlockingWarnings, finding)
		}
	}

	var completeIdentity *researchidentity.BoundResearchIdentity
	if identityAssessment.Status == researchidentity.StatusComplete {
		completeIdentity = identityAssessment.Identity
	}
	eligible := completeIdentity != nil && localIntegrity == LocalIntegrityPassed && input.Classification == ResearchStatusValidatedResearchLead
	candidateResult := ResearchCandidateResult{
		Classification:   input.Classification,
		ObservationCount: len(timestamps),
	}
	if completeIdentity != nil {
		candidateResult.CandidateID = completeIdentity.Candidate.CandidateID
		candidateResult.CandidateVersion = completeIdentity.Candidate.CandidateVersion
	}
	sortFindings(identityContractFindings)
	sortFindings(blockingFindings)
	sortFindings(nonBlockingWarnings)

	var evidence *ResearchEvidence
	if completeIdentity != nil && metricsErr == nil {
		metricResults, err := buildMetricResults(*completeIdentity, metrics, config.MetricRiskFreeRate, config.MetricPeriodsPerYear, len(returns))
		if err != nil {
			result.Failure = DiagnosticsFailureSerialization
			return result, fmt.Errorf("%w: %v", ErrResearchDiagnosticsSerialization, err)
		}
		builtEvidence, err := buildResearchEvidence(*completeIdentity, metricResults, input.Classification, eligible, blockingFindings, nonBlockingWarnings)
		if err != nil {
			result.Failure = DiagnosticsFailureSerialization
			return result, fmt.Errorf("%w: %v", ErrResearchDiagnosticsSerialization, err)
		}
		evidence = &builtEvidence
	}
	diagnostic := LocalResearchDiagnostics{
		Contract:             canonicalcontract.NewHeader(LocalResearchDiagnosticsSchemaName, LocalResearchDiagnosticsSchemaVersion, LocalResearchDiagnosticsArtifactRole),
		AuthorityStatus:      AuthorityStatusNoneResearchOnly,
		IdentityStatus:       identityAssessment.Status,
		EligibleForRIFReview: eligible,
		CandidateResult:      candidateResult,
		IdentityFindings:     identityContractFindings,
		LocalIntegrity:       localIntegrity,
		BlockingFindings:     blockingFindings,
		NonBlockingWarnings:  nonBlockingWarnings,
		ResearchEvidence:     evidence,
	}
	var err error
	diagnostic.ArtifactHash, err = contractArtifactHash(LocalResearchDiagnosticsSchemaName, LocalResearchDiagnosticsArtifactRole, diagnostic)
	if err != nil {
		result.Failure = DiagnosticsFailureSerialization
		return result, fmt.Errorf("%w: %v", ErrResearchDiagnosticsSerialization, err)
	}

	path := input.Stem + ".research_diagnostics.json"
	result = ResearchDiagnosticsResult{
		ArtifactDisposition: ArtifactSuppressed,
		Failure:             DiagnosticsFailureNone,
		IdentityStatus:      identityAssessment.Status,
		IdentityFindings:    append([]researchidentity.Finding(nil), identityAssessment.Findings...),
		EligibleForReview:   eligible,
		LocalIntegrity:      localIntegrity,
		BlockingFindings:    len(blockingFindings),
		NonBlockingWarnings: len(nonBlockingWarnings),
	}
	if err := writeResearchDiagnosticsFile(path, diagnostic); err != nil {
		switch {
		case errors.Is(err, ErrResearchDiagnosticsSerialization):
			result.Failure = DiagnosticsFailureSerialization
		case errors.Is(err, ErrResearchDiagnosticsPersistence):
			result.Failure = DiagnosticsFailurePersistence
		default:
			result.Failure = DiagnosticsFailurePersistence
		}
		return result, err
	}

	result.ArtifactDisposition = ArtifactEmitted
	result.ArtifactPath = path
	if identityErr != nil {
		result.Failure = DiagnosticsFailureIdentityDerivation
		return result, identityErr
	}
	return result, nil
}

func validateBridgeMetricSeries(returns []float64, timestamps []int64) error {
	if len(returns) == 0 || len(timestamps) == 0 || len(returns) != len(timestamps) {
		return fmt.Errorf("metric return/timestamp series is empty or count-inconsistent")
	}
	for i, value := range returns {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("metric return %d is non-finite", i)
		}
	}
	return nil
}

func normalizeIdentityAssessment(assessment researchidentity.Assessment, err error) (researchidentity.Assessment, error) {
	if !knownIdentityStatus(assessment.Status) {
		reason := fmt.Errorf("unknown identity status %q", assessment.Status)
		return researchidentity.Assessment{
			Status: researchidentity.StatusValidationFailed,
			Findings: []researchidentity.Finding{{
				Code: "UNKNOWN_IDENTITY_STATUS", Domain: "identity_status", Reason: reason.Error(),
				Status: researchidentity.StatusValidationFailed, Blocking: true,
			}},
		}, &researchidentity.DerivationError{Status: researchidentity.StatusValidationFailed, Code: "UNKNOWN_IDENTITY_STATUS", Err: reason}
	}
	if assessment.Status == researchidentity.StatusComplete {
		if assessment.Identity == nil || err != nil || hasBlockingIdentityFinding(assessment.Findings) {
			reason := fmt.Errorf("complete identity result has missing identity, error, or blocking finding")
			return researchidentity.Assessment{
				Status: researchidentity.StatusValidationFailed,
				Findings: []researchidentity.Finding{{
					Code: "INVALID_COMPLETE_IDENTITY_RESULT", Domain: "identity_result", Reason: reason.Error(),
					Status: researchidentity.StatusValidationFailed, Blocking: true,
				}},
			}, &researchidentity.DerivationError{Status: researchidentity.StatusValidationFailed, Code: "INVALID_COMPLETE_IDENTITY_RESULT", Err: reason}
		}
		return assessment, nil
	}
	assessment.Identity = nil
	if err == nil {
		reason := fmt.Errorf("incomplete identity status %s returned without derivation error", assessment.Status)
		err = &researchidentity.DerivationError{Status: assessment.Status, Code: "INCOMPLETE_IDENTITY_WITHOUT_ERROR", Err: reason}
		assessment.Findings = append(assessment.Findings, researchidentity.Finding{
			Code: "INCOMPLETE_IDENTITY_WITHOUT_ERROR", Domain: "identity_result", Reason: reason.Error(), Status: assessment.Status, Blocking: true,
		})
	}
	return assessment, err
}

func knownIdentityStatus(status researchidentity.IdentityStatus) bool {
	switch status {
	case researchidentity.StatusComplete,
		researchidentity.StatusCandidateIncomplete,
		researchidentity.StatusConfigurationMissing,
		researchidentity.StatusDirtyEngineSource,
		researchidentity.StatusDatasetIncomplete,
		researchidentity.StatusPITIncomplete,
		researchidentity.StatusFeatureIncomplete,
		researchidentity.StatusRegimeIncomplete,
		researchidentity.StatusConsumedIncomplete,
		researchidentity.StatusSeriesIncomplete,
		researchidentity.StatusConflict,
		researchidentity.StatusValidationFailed:
		return true
	default:
		return false
	}
}

func hasBlockingIdentityFinding(findings []researchidentity.Finding) bool {
	for _, finding := range findings {
		if finding.Blocking {
			return true
		}
	}
	return false
}

func identityFindings(findings []researchidentity.Finding) []researchidentity.Finding {
	return append([]researchidentity.Finding{}, findings...)
}

func validateResearchAssessment(input ResearchAssessment) error {
	if strings.TrimSpace(input.Stem) == "" {
		return fmt.Errorf("%w: stem is required", ErrInvalidResearchInput)
	}
	if !isKnownResearchClassification(input.Classification) {
		return fmt.Errorf("%w: unknown research classification %q", ErrInvalidResearchInput, input.Classification)
	}
	derivedClassification, err := classificationFromGates(input.ClassificationGates)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidResearchInput, err)
	}
	if input.Classification != derivedClassification {
		return fmt.Errorf("%w: classification %q disagrees with gate-derived %q", ErrInvalidResearchInput, input.Classification, derivedClassification)
	}
	if input.ExecutionSeriesGeneration != executionseries.GenerationVersion {
		return fmt.Errorf("%w: execution series generation %q is not canonical %q", ErrInvalidResearchInput, input.ExecutionSeriesGeneration, executionseries.GenerationVersion)
	}
	return nil
}

func classificationFromGates(gates []ClassificationGate) (string, error) {
	if len(gates) == 0 {
		return "", fmt.Errorf("classification gates are required")
	}
	seen := make(map[string]struct{}, len(gates))
	failed := 0
	criticalFailed := false
	seriesGatePassed := false
	for _, gate := range gates {
		name := strings.TrimSpace(gate.Name)
		if name == "" {
			return "", fmt.Errorf("classification gate name is required")
		}
		if _, exists := seen[name]; exists {
			return "", fmt.Errorf("duplicate classification gate %q", name)
		}
		seen[name] = struct{}{}
		if name == "execution_series_identity" && gate.Passed && gate.Critical {
			seriesGatePassed = true
		}
		if !gate.Passed {
			failed++
			criticalFailed = criticalFailed || gate.Critical
		}
	}
	if !seriesGatePassed {
		return "", fmt.Errorf("passing critical execution_series_identity gate is required")
	}
	if failed == 0 {
		return ResearchStatusValidatedResearchLead, nil
	}
	if criticalFailed {
		return ResearchStatusRejected, nil
	}
	if failed <= 2 {
		return ResearchStatusFragile, nil
	}
	return ResearchStatusRejected, nil
}

func isKnownResearchClassification(classification string) bool {
	switch classification {
	case ResearchStatusValidatedResearchLead,
		ResearchStatusFragile,
		ResearchStatusNeedsMoreData,
		ResearchStatusRejected:
		return true
	default:
		return false
	}
}

func newResearchFinding(code, _ string, reason string, blocking bool, _ string) researchidentity.Finding {
	return researchidentity.Finding{
		Code: code, Domain: "local_integrity", Reason: reason,
		Status: researchidentity.StatusValidationFailed, Blocking: blocking,
	}
}

func sortFindings(findings []researchidentity.Finding) {
	sort.Slice(findings, func(i, j int) bool { return findings[i].Code < findings[j].Code })
}
