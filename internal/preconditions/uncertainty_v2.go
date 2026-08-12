package preconditions

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
)

const (
	AcceptedUncertaintyMethodVersion = "ak.engine.uncertainty.cluster-bootstrap.v2"
	UncertaintyStatusAccepted        = "ACCEPTED"
	UncertaintyIntervalNotReportable = "UNCERTAINTY_INTERVAL_NOT_REPORTABLE"
	UncertaintyReportableSampleFail  = "UNCERTAINTY_INTERVAL_REPORTABLE_BUT_SAMPLE_GATE_FAILS"
	UncertaintyQualificationEligible = "UNCERTAINTY_INTERVAL_QUALIFICATION_ELIGIBLE"
)

type AcceptedUncertaintyMethod struct {
	Version                 string  `json:"version"`
	Status                  string  `json:"status"`
	Estimand                string  `json:"estimand"`
	PrimaryStatistic        string  `json:"primary_statistic"`
	QualificationRule       string  `json:"qualification_rule"`
	ConfidenceLevel         float64 `json:"confidence_level"`
	LowerPercentile         float64 `json:"lower_percentile"`
	SamplingUnit            string  `json:"sampling_unit"`
	ResamplingMethod        string  `json:"resampling_method"`
	NumberOfResamples       int     `json:"number_of_resamples"`
	DrawCountRule           string  `json:"draw_count_rule"`
	QuantileIndexRule       string  `json:"quantile_index_rule"`
	StratificationRule      string  `json:"stratification_rule"`
	SeedDerivationRule      string  `json:"seed_derivation_rule"`
	MinimumSampleRule       string  `json:"minimum_sample_rule"`
	InvalidDataRule         string  `json:"invalid_data_rule"`
	DeterministicOutputRule string  `json:"deterministic_output_rule"`
}

type CanonicalIdentityBinding struct {
	Identity string `json:"identity"`
	Hash     string `json:"hash"`
}

type BootstrapSeedIdentityV2 struct {
	SchemaVersion           string                   `json:"schema_version"`
	UncertaintyContractHash string                   `json:"uncertainty_contract_hash"`
	IndependencePolicyHash  string                   `json:"independence_policy_hash"`
	FrozenCandidate         CanonicalIdentityBinding `json:"frozen_candidate"`
	Dataset                 CanonicalIdentityBinding `json:"dataset"`
	Manifest                CanonicalIdentityBinding `json:"manifest"`
	Partition               CanonicalIdentityBinding `json:"partition"`
	CostModel               CanonicalIdentityBinding `json:"cost_model"`
}

type UncertaintyResultV2 struct {
	SchemaVersion     string   `json:"schema_version"`
	MethodVersion     string   `json:"method_version"`
	MethodHash        string   `json:"method_hash"`
	MethodStatus      string   `json:"method_status"`
	SampleStatus      string   `json:"sample_status"`
	ClusterCount      int      `json:"cluster_count"`
	Estimator         float64  `json:"estimator"`
	LowerBound        *float64 `json:"lower_bound,omitempty"`
	ConfidenceLevel   float64  `json:"confidence_level"`
	QuantileIndex     *int     `json:"quantile_index,omitempty"`
	NumberOfResamples int      `json:"number_of_resamples"`
	SeedHex           string   `json:"seed_hex"`
	SeedEvidenceHash  string   `json:"seed_evidence_hash"`
	QualificationPass bool     `json:"qualification_pass"`
	EvidenceHash      string   `json:"evidence_hash"`
}

func AcceptedUncertaintyMethodV2() AcceptedUncertaintyMethod {
	return AcceptedUncertaintyMethod{
		Version: AcceptedUncertaintyMethodVersion, Status: UncertaintyStatusAccepted,
		Estimand:          "mean mandatory-cost net expectancy per accepted independent cluster",
		PrimaryStatistic:  "one-sided 95% lower confidence bound for mean net expectancy",
		QualificationRule: "qualification requires an eligible sample and lower confidence bound strictly greater than zero",
		ConfidenceLevel:   0.95, LowerPercentile: 0.05,
		SamplingUnit:            "one deterministic net-return observation per accepted independent cluster; raw events are prohibited",
		ResamplingMethod:        "nonparametric percentile cluster bootstrap with replacement",
		NumberOfResamples:       10000,
		DrawCountRule:           "for N observations, every replicate draws exactly N clusters independently with replacement and calculates their arithmetic mean",
		QuantileIndexRule:       "sort replicate means by deterministic numeric ascending order; nearest-rank fifth percentile uses zero-based index ceil(0.05*B)-1 with no interpolation; B=10000 gives index 499",
		StratificationRule:      "none: no symbol, month, quarter, regime, or performance stratification or rebalancing",
		SeedDerivationRule:      "SHA-256 canonical JSON of contract hashes plus frozen candidate, dataset, manifest, partition, and cost-model identity/hash bindings; first eight digest bytes big-endian seed SplitMix64",
		MinimumSampleRule:       "N<30 not reportable; 30<=N<300 reportable but sample gate fails; N>=300 qualification eligible",
		InvalidDataRule:         "empty input, duplicate cluster IDs, non-finite returns, missing identities, identity/hash mismatch, malformed canonical serialization, contract/hash mismatch, and unsupported versions fail closed; never silently drop observations",
		DeterministicOutputRule: "sort observations by cluster ID, use SplitMix64 and explicit quantile indexing, and hash canonical result bytes; equal complete input produces byte-equivalent output",
	}
}

func AcceptedUncertaintyMethodHashV2(method AcceptedUncertaintyMethod) (string, error) {
	if err := validateAcceptedUncertaintyMethodV2(method); err != nil {
		return "", err
	}
	return canonicalDigest(method)
}

func BindCanonicalIdentityV2(identity string) (CanonicalIdentityBinding, error) {
	if strings.TrimSpace(identity) == "" || strings.TrimSpace(identity) != identity {
		return CanonicalIdentityBinding{}, errors.New("canonical identity is required without surrounding whitespace")
	}
	hash, err := canonicalDigest(struct {
		Identity string `json:"identity"`
	}{identity})
	if err != nil {
		return CanonicalIdentityBinding{}, err
	}
	return CanonicalIdentityBinding{Identity: identity, Hash: hash}, nil
}

func EncodeBootstrapSeedIdentityV2(identity BootstrapSeedIdentityV2) ([]byte, error) {
	if err := validateBootstrapSeedIdentityV2(identity, ""); err != nil {
		return nil, err
	}
	data, err := json.Marshal(identity)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func DecodeBootstrapSeedIdentityV2(data []byte) (BootstrapSeedIdentityV2, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var identity BootstrapSeedIdentityV2
	if err := decoder.Decode(&identity); err != nil {
		return BootstrapSeedIdentityV2{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return BootstrapSeedIdentityV2{}, errors.New("bootstrap seed identity has trailing JSON data")
	}
	if err := validateBootstrapSeedIdentityV2(identity, ""); err != nil {
		return BootstrapSeedIdentityV2{}, err
	}
	canonical, err := EncodeBootstrapSeedIdentityV2(identity)
	if err != nil {
		return BootstrapSeedIdentityV2{}, err
	}
	if !bytes.Equal(data, canonical) {
		return BootstrapSeedIdentityV2{}, errors.New("bootstrap seed identity is not canonical JSON serialization")
	}
	return identity, nil
}

func EstimateLowerBoundV2(observations []ClusterObservation, method AcceptedUncertaintyMethod, identity BootstrapSeedIdentityV2) (UncertaintyResultV2, error) {
	methodHash, err := AcceptedUncertaintyMethodHashV2(method)
	if err != nil {
		return UncertaintyResultV2{}, err
	}
	if err := validateBootstrapSeedIdentityV2(identity, methodHash); err != nil {
		return UncertaintyResultV2{}, err
	}
	if len(observations) == 0 {
		return UncertaintyResultV2{}, errors.New("empty cluster input")
	}
	valuesByID := make(map[string]float64, len(observations))
	for _, observation := range observations {
		if strings.TrimSpace(observation.ClusterID) == "" || !finite(observation.NetValue) {
			return UncertaintyResultV2{}, errors.New("cluster identity and finite net return are required")
		}
		if _, duplicate := valuesByID[observation.ClusterID]; duplicate {
			return UncertaintyResultV2{}, fmt.Errorf("duplicate cluster identity %q", observation.ClusterID)
		}
		valuesByID[observation.ClusterID] = observation.NetValue
	}
	ids := make([]string, 0, len(valuesByID))
	for id := range valuesByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	values := make([]float64, len(ids))
	total := 0.0
	for i, id := range ids {
		values[i] = valuesByID[id]
		total += values[i]
	}
	estimator := total / float64(len(values))
	seed, seedHash, err := DeriveBootstrapSeedV2(identity, methodHash)
	if err != nil {
		return UncertaintyResultV2{}, err
	}
	result := UncertaintyResultV2{
		SchemaVersion: "ak.engine.uncertainty-result.cluster-bootstrap.v2", MethodVersion: method.Version,
		MethodHash: methodHash, MethodStatus: method.Status, ClusterCount: len(values), Estimator: estimator,
		ConfidenceLevel: method.ConfidenceLevel, NumberOfResamples: method.NumberOfResamples,
		SeedHex: fmt.Sprintf("%016x", seed), SeedEvidenceHash: seedHash,
	}
	switch {
	case len(values) < 30:
		result.SampleStatus = UncertaintyIntervalNotReportable
		result.NumberOfResamples = 0
	case len(values) < 300:
		result.SampleStatus = UncertaintyReportableSampleFail
	default:
		result.SampleStatus = UncertaintyQualificationEligible
	}
	if len(values) >= 30 {
		replicates := make([]float64, method.NumberOfResamples)
		state := seed
		for sample := range replicates {
			sum := 0.0
			for range values {
				state = splitMix64(state)
				sum += values[int(state%uint64(len(values)))]
			}
			replicates[sample] = sum / float64(len(values))
		}
		sort.Float64s(replicates)
		index, err := QuantileIndexV2(method.LowerPercentile, len(replicates))
		if err != nil {
			return UncertaintyResultV2{}, err
		}
		bound := replicates[index]
		result.LowerBound = &bound
		result.QuantileIndex = &index
		result.QualificationPass = result.SampleStatus == UncertaintyQualificationEligible && bound > 0
	}
	hash, err := uncertaintyResultHashV2(result)
	if err != nil {
		return UncertaintyResultV2{}, err
	}
	result.EvidenceHash = hash
	return result, nil
}

func QuantileIndexV2(percentile float64, count int) (int, error) {
	if !finite(percentile) || percentile <= 0 || percentile > 1 || count <= 0 {
		return 0, errors.New("valid percentile and positive count are required")
	}
	index := int(math.Ceil(percentile*float64(count))) - 1
	if index < 0 || index >= count {
		return 0, errors.New("quantile index is outside sorted estimates")
	}
	return index, nil
}

func DeriveBootstrapSeedV2(identity BootstrapSeedIdentityV2, methodHash string) (uint64, string, error) {
	if err := validateBootstrapSeedIdentityV2(identity, methodHash); err != nil {
		return 0, "", err
	}
	data, err := json.Marshal(identity)
	if err != nil {
		return 0, "", err
	}
	digest := sha256.Sum256(data)
	return binary.BigEndian.Uint64(digest[:8]), "sha256:" + hex.EncodeToString(digest[:]), nil
}

func EncodeUncertaintyResultV2(result UncertaintyResultV2) ([]byte, error) {
	want, err := uncertaintyResultHashV2(result)
	if err != nil {
		return nil, err
	}
	if !validSHA256(result.EvidenceHash) || result.EvidenceHash != want {
		return nil, errors.New("uncertainty result evidence hash mismatch")
	}
	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func uncertaintyResultHashV2(result UncertaintyResultV2) (string, error) {
	copyResult := result
	copyResult.EvidenceHash = ""
	return canonicalDigest(copyResult)
}

func validateAcceptedUncertaintyMethodV2(method AcceptedUncertaintyMethod) error {
	want := AcceptedUncertaintyMethodV2()
	if method != want {
		return errors.New("unsupported method version or accepted contract mutation")
	}
	return nil
}

func validateBootstrapSeedIdentityV2(identity BootstrapSeedIdentityV2, methodHash string) error {
	if identity.SchemaVersion != "ak.engine.bootstrap-seed-identity.v2" {
		return errors.New("unsupported bootstrap seed identity version")
	}
	if !validSHA256(identity.UncertaintyContractHash) || !validSHA256(identity.IndependencePolicyHash) {
		return errors.New("valid uncertainty and independence contract hashes are required")
	}
	if methodHash != "" && identity.UncertaintyContractHash != methodHash {
		return errors.New("uncertainty contract identity/hash mismatch")
	}
	for name, binding := range map[string]CanonicalIdentityBinding{
		"frozen candidate": identity.FrozenCandidate, "dataset": identity.Dataset, "manifest": identity.Manifest,
		"partition": identity.Partition, "cost model": identity.CostModel,
	} {
		want, err := BindCanonicalIdentityV2(binding.Identity)
		if err != nil || binding.Hash != want.Hash {
			return fmt.Errorf("%s identity/hash mismatch", name)
		}
	}
	return nil
}
