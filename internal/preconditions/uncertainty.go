package preconditions

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

const UncertaintyMethodVersion = "ak.engine.uncertainty.cluster-bootstrap.v1"

type UncertaintyMethod struct {
	Version                 string  `json:"version"`
	Status                  string  `json:"status"`
	Estimator               string  `json:"estimator"`
	ConfidenceLevel         float64 `json:"confidence_level"`
	SamplingUnit            string  `json:"sampling_unit"`
	ResamplingMethod        string  `json:"resampling_method"`
	BlockConstruction       string  `json:"block_construction"`
	Seed                    uint64  `json:"seed"`
	NumberOfResamples       int     `json:"number_of_resamples"`
	IntervalType            string  `json:"interval_type"`
	TreatmentOfCosts        string  `json:"treatment_of_costs"`
	InvalidObservationRule  string  `json:"invalid_observation_rule"`
	DeterministicOutputRule string  `json:"deterministic_output_rule"`
}

type ClusterObservation struct {
	ClusterID string  `json:"cluster_id"`
	NetValue  float64 `json:"net_value"`
}

type UncertaintyResult struct {
	MethodVersion     string  `json:"method_version"`
	MethodHash        string  `json:"method_hash"`
	MethodStatus      string  `json:"method_status"`
	ClusterCount      int     `json:"cluster_count"`
	Estimator         float64 `json:"estimator"`
	LowerBound        float64 `json:"lower_bound"`
	ConfidenceLevel   float64 `json:"confidence_level"`
	Seed              uint64  `json:"seed"`
	NumberOfResamples int     `json:"number_of_resamples"`
}

func ProposedUncertaintyMethod() UncertaintyMethod {
	return UncertaintyMethod{
		Version: UncertaintyMethodVersion, Status: PolicyStatusProposedNotAccepted,
		Estimator:       "arithmetic mean of one frozen-cost net observation per independent cluster",
		ConfidenceLevel: 0.95, SamplingUnit: "independent cluster",
		ResamplingMethod:  "nonparametric cluster bootstrap with replacement",
		BlockConstruction: "one complete independent cluster is one indivisible block",
		Seed:              0x5052344230523150, NumberOfResamples: 4096,
		IntervalType:            "one-sided lower percentile bound",
		TreatmentOfCosts:        "input observations must already apply the frozen declared cost vector",
		InvalidObservationRule:  "missing IDs, non-finite values, and conflicting duplicate cluster IDs fail closed; exact duplicates are deduplicated",
		DeterministicOutputRule: "sort by cluster ID; SplitMix64 PRNG; ascending replicate means; floor((1-confidence)*B) index",
	}
}

func UncertaintyMethodHash(method UncertaintyMethod) (string, error) {
	if err := validateUncertaintyMethod(method); err != nil {
		return "", err
	}
	return canonicalDigest(method)
}

func EstimateLowerBound(observations []ClusterObservation, method UncertaintyMethod) (UncertaintyResult, error) {
	if err := validateUncertaintyMethod(method); err != nil {
		return UncertaintyResult{}, err
	}
	unique := map[string]float64{}
	for _, observation := range observations {
		if strings.TrimSpace(observation.ClusterID) == "" || !finite(observation.NetValue) {
			return UncertaintyResult{}, errors.New("invalid cluster observation")
		}
		if prior, exists := unique[observation.ClusterID]; exists {
			if math.Float64bits(prior) != math.Float64bits(observation.NetValue) {
				return UncertaintyResult{}, fmt.Errorf("conflicting duplicate cluster %s", observation.ClusterID)
			}
			continue
		}
		unique[observation.ClusterID] = observation.NetValue
	}
	if len(unique) < 2 {
		return UncertaintyResult{}, errors.New("at least two independent clusters are required")
	}
	ids := make([]string, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	values := make([]float64, len(ids))
	sum := 0.0
	for i, id := range ids {
		values[i] = unique[id]
		sum += values[i]
	}
	estimator := sum / float64(len(values))
	replicates := make([]float64, method.NumberOfResamples)
	state := method.Seed
	for sample := range replicates {
		total := 0.0
		for range values {
			state = splitMix64(state)
			total += values[int(state%uint64(len(values)))]
		}
		replicates[sample] = total / float64(len(values))
	}
	sort.Float64s(replicates)
	index := int(math.Floor((1 - method.ConfidenceLevel) * float64(method.NumberOfResamples)))
	if index < 0 {
		index = 0
	}
	if index >= len(replicates) {
		index = len(replicates) - 1
	}
	hash, _ := UncertaintyMethodHash(method)
	return UncertaintyResult{
		MethodVersion: method.Version, MethodHash: hash, MethodStatus: method.Status, ClusterCount: len(values),
		Estimator: estimator, LowerBound: replicates[index], ConfidenceLevel: method.ConfidenceLevel,
		Seed: method.Seed, NumberOfResamples: method.NumberOfResamples,
	}, nil
}

func validateUncertaintyMethod(method UncertaintyMethod) error {
	if method.Version != UncertaintyMethodVersion || method.Status != PolicyStatusProposedNotAccepted {
		return errors.New("uncertainty method must remain explicitly proposed")
	}
	if method.ConfidenceLevel <= 0.5 || method.ConfidenceLevel >= 1 || method.NumberOfResamples < 100 || method.Seed == 0 {
		return errors.New("invalid uncertainty parameters")
	}
	for name, value := range map[string]string{"estimator": method.Estimator, "sampling unit": method.SamplingUnit, "method": method.ResamplingMethod, "block construction": method.BlockConstruction, "interval": method.IntervalType, "cost treatment": method.TreatmentOfCosts, "invalid rule": method.InvalidObservationRule, "output rule": method.DeterministicOutputRule} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	return nil
}

func splitMix64(value uint64) uint64 {
	value += 0x9e3779b97f4a7c15
	value = (value ^ (value >> 30)) * 0xbf58476d1ce4e5b9
	value = (value ^ (value >> 27)) * 0x94d049bb133111eb
	return value ^ (value >> 31)
}
