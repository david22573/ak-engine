package preconditions

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestAcceptedUncertaintyV2PositiveNegativeZeroAndSkew(t *testing.T) {
	method := AcceptedUncertaintyMethodV2()
	identity := syntheticSeedIdentityV2(t, method)
	positive := syntheticClusterObservationsV2(300, func(int) float64 { return 2 })
	pos, err := EstimateLowerBoundV2(positive, method, identity)
	if err != nil {
		t.Fatal(err)
	}
	if pos.SampleStatus != UncertaintyQualificationEligible || pos.LowerBound == nil || *pos.LowerBound <= 0 || !pos.QualificationPass {
		t.Fatalf("clearly positive synthetic result=%+v", pos)
	}
	for name, values := range map[string][]ClusterObservation{
		"negative": syntheticClusterObservationsV2(30, func(int) float64 { return -2 }),
		"zero":     syntheticClusterObservationsV2(30, func(int) float64 { return 0 }),
		"skewed": syntheticClusterObservationsV2(30, func(i int) float64 {
			if i == 29 {
				return 100
			}
			return -1
		}),
		"outlier": syntheticClusterObservationsV2(30, func(i int) float64 {
			if i == 0 {
				return 1000
			}
			return 1
		}),
	} {
		t.Run(name, func(t *testing.T) {
			result, err := EstimateLowerBoundV2(values, method, identity)
			if err != nil || result.LowerBound == nil || !finite(*result.LowerBound) {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if result.QualificationPass {
				t.Fatal("sub-300 sample qualified")
			}
			if name == "negative" && *result.LowerBound >= 0 {
				t.Fatal("negative synthetic expectancy has nonnegative lower bound")
			}
			if name == "zero" && *result.LowerBound != 0 {
				t.Fatal("zero synthetic expectancy did not produce zero bound")
			}
		})
	}
}

func TestAcceptedUncertaintyV2SampleBoundaries(t *testing.T) {
	method := AcceptedUncertaintyMethodV2()
	identity := syntheticSeedIdentityV2(t, method)
	tests := []struct {
		n      int
		status string
		hasLCB bool
	}{
		{29, UncertaintyIntervalNotReportable, false},
		{30, UncertaintyReportableSampleFail, true},
		{299, UncertaintyReportableSampleFail, true},
		{300, UncertaintyQualificationEligible, true},
	}
	for _, test := range tests {
		result, err := EstimateLowerBoundV2(syntheticClusterObservationsV2(test.n, func(int) float64 { return 1 }), method, identity)
		if err != nil {
			t.Fatal(err)
		}
		if result.SampleStatus != test.status || (result.LowerBound != nil) != test.hasLCB {
			t.Fatalf("N=%d status=%s bound=%v", test.n, result.SampleStatus, result.LowerBound)
		}
	}
}

func TestAcceptedUncertaintyV2OrderingSeedAndBytes(t *testing.T) {
	method := AcceptedUncertaintyMethodV2()
	identity := syntheticSeedIdentityV2(t, method)
	forward := syntheticClusterObservationsV2(30, func(i int) float64 { return float64(i - 10) })
	reverse := append([]ClusterObservation(nil), forward...)
	for left, right := 0, len(reverse)-1; left < right; left, right = left+1, right-1 {
		reverse[left], reverse[right] = reverse[right], reverse[left]
	}
	first, err := EstimateLowerBoundV2(forward, method, identity)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EstimateLowerBoundV2(reverse, method, identity)
	if err != nil {
		t.Fatal(err)
	}
	firstBytes, err := EncodeUncertaintyResultV2(first)
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := EncodeUncertaintyResultV2(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatal("reordered observations or repeated execution changed bytes")
	}
	seedA, evidenceA, err := DeriveBootstrapSeedV2(identity, first.MethodHash)
	if err != nil {
		t.Fatal(err)
	}
	seedB, evidenceB, err := DeriveBootstrapSeedV2(identity, first.MethodHash)
	if err != nil || seedA != seedB || evidenceA != evidenceB {
		t.Fatal("identical complete identity changed seed")
	}
	for name, mutate := range map[string]func(*BootstrapSeedIdentityV2){
		"candidate": func(value *BootstrapSeedIdentityV2) {
			value.FrozenCandidate = mustBindV2(t, "synthetic-candidate-mutated")
		},
		"dataset":   func(value *BootstrapSeedIdentityV2) { value.Dataset = mustBindV2(t, "synthetic-dataset-mutated") },
		"partition": func(value *BootstrapSeedIdentityV2) { value.Partition = mustBindV2(t, "synthetic-partition-mutated") },
	} {
		t.Run(name, func(t *testing.T) {
			mutated := identity
			mutate(&mutated)
			seed, evidence, err := DeriveBootstrapSeedV2(mutated, first.MethodHash)
			if err != nil || seed == seedA || evidence == evidenceA {
				t.Fatalf("identity mutation seed=%x evidence=%s err=%v", seed, evidence, err)
			}
		})
	}
}

func TestAcceptedUncertaintyV2InvalidDataFailsClosed(t *testing.T) {
	method := AcceptedUncertaintyMethodV2()
	identity := syntheticSeedIdentityV2(t, method)
	valid := syntheticClusterObservationsV2(30, func(int) float64 { return 1 })
	invalidSets := [][]ClusterObservation{
		nil,
		append(append([]ClusterObservation(nil), valid...), valid[0]),
		{{ClusterID: "nan", NetValue: math.NaN()}},
		{{ClusterID: "inf", NetValue: math.Inf(1)}},
	}
	for _, observations := range invalidSets {
		if _, err := EstimateLowerBoundV2(observations, method, identity); err == nil {
			t.Fatal("invalid observations passed")
		}
	}
	for name, mutate := range map[string]func(*BootstrapSeedIdentityV2){
		"cost":              func(value *BootstrapSeedIdentityV2) { value.CostModel = CanonicalIdentityBinding{} },
		"dataset":           func(value *BootstrapSeedIdentityV2) { value.Dataset = CanonicalIdentityBinding{} },
		"partition":         func(value *BootstrapSeedIdentityV2) { value.Partition = CanonicalIdentityBinding{} },
		"hash mismatch":     func(value *BootstrapSeedIdentityV2) { value.Dataset.Hash = value.Manifest.Hash },
		"contract mismatch": func(value *BootstrapSeedIdentityV2) { value.UncertaintyContractHash = value.IndependencePolicyHash },
	} {
		t.Run(name, func(t *testing.T) {
			mutated := identity
			mutate(&mutated)
			if _, err := EstimateLowerBoundV2(valid, method, mutated); err == nil {
				t.Fatal("invalid identity passed")
			}
		})
	}
	mutatedMethod := method
	mutatedMethod.NumberOfResamples = 9999
	if _, err := EstimateLowerBoundV2(valid, mutatedMethod, identity); err == nil {
		t.Fatal("unsupported method mutation passed")
	}
}

func TestAcceptedUncertaintyV2CanonicalSerializationAndQuantile(t *testing.T) {
	method := AcceptedUncertaintyMethodV2()
	identity := syntheticSeedIdentityV2(t, method)
	encoded, err := EncodeBootstrapSeedIdentityV2(identity)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeBootstrapSeedIdentityV2(encoded)
	if err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(identity)
	right, _ := json.Marshal(decoded)
	if string(left) != string(right) {
		t.Fatal("canonical identity round trip changed bytes")
	}
	malformed := append([]byte(" "), encoded...)
	if _, err := DecodeBootstrapSeedIdentityV2(malformed); err == nil {
		t.Fatal("noncanonical serialization passed")
	}
	index, err := QuantileIndexV2(0.05, 10000)
	if err != nil || index != 499 {
		t.Fatalf("quantile index=%d err=%v", index, err)
	}
	originalHash, err := AcceptedUncertaintyMethodHashV2(method)
	if err != nil {
		t.Fatal(err)
	}
	mutated := method
	mutated.QuantileIndexRule += "; mutated"
	mutatedHash, err := canonicalDigest(mutated)
	if err != nil || originalHash == mutatedHash {
		t.Fatal("contract mutation did not change canonical hash")
	}
}

func TestAcceptedUncertaintyV2RawEventDuplicationDoesNotChangeClusterInput(t *testing.T) {
	event := syntheticEvent(t, "AAAUSDT", syntheticTimeV2())
	clusters, err := ClusterEventsV2([]RetainedEvent{event, event}, RevisedIndependencePolicyV2())
	if err != nil || len(clusters) != 1 || len(clusters[0].MemberEventIDs) != 1 {
		t.Fatalf("raw duplication changed clusters: %+v err=%v", clusters, err)
	}
	result, err := AggregateClusterNetReturnV2(clusters[0], []MemberNetReturn{{EventID: event.EventID, NetValue: 1}})
	if err != nil || result.ClusterID != clusters[0].ClusterID {
		t.Fatalf("cluster observation mismatch: %+v err=%v", result, err)
	}
}

func syntheticSeedIdentityV2(t *testing.T, method AcceptedUncertaintyMethod) BootstrapSeedIdentityV2 {
	t.Helper()
	methodHash, err := AcceptedUncertaintyMethodHashV2(method)
	if err != nil {
		t.Fatal(err)
	}
	return BootstrapSeedIdentityV2{
		SchemaVersion: "ak.engine.bootstrap-seed-identity.v2", UncertaintyContractHash: methodHash,
		IndependencePolicyHash: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		FrozenCandidate:        mustBindV2(t, "synthetic-frozen-candidate-v1"), Dataset: mustBindV2(t, "synthetic-dataset-v1"),
		Manifest: mustBindV2(t, "synthetic-manifest-v1"), Partition: mustBindV2(t, "synthetic-partition-v1"), CostModel: mustBindV2(t, "synthetic-cost-model-v1"),
	}
}

func mustBindV2(t *testing.T, identity string) CanonicalIdentityBinding {
	t.Helper()
	binding, err := BindCanonicalIdentityV2(identity)
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func syntheticClusterObservationsV2(count int, value func(int) float64) []ClusterObservation {
	observations := make([]ClusterObservation, count)
	for i := range observations {
		observations[i] = ClusterObservation{ClusterID: string(rune('a'+i%26)) + "-synthetic-" + fmtIntV2(i), NetValue: value(i)}
	}
	return observations
}

func fmtIntV2(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	buffer := [20]byte{}
	position := len(buffer)
	for value > 0 {
		position--
		buffer[position] = digits[value%10]
		value /= 10
	}
	return string(buffer[position:])
}

func syntheticTimeV2() time.Time {
	return time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC)
}
