package preconditions

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"
)

func TestCountAlternativeSymbolThresholdBoundaries(t *testing.T) {
	policy := GovernanceConcentrationAlternatives()[0]
	for _, test := range []struct {
		name       string
		dominant   int
		want       float64
		wantPassed bool
	}{
		{"exactly below", 49, 49, true},
		{"exactly at", 50, 50, true},
		{"minimally above", 51, 51, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			rows := countSymbolFixture(t, test.dominant)
			got := evaluateSyntheticAlternative(t, policy, rows)
			metric := got.Partitions[0].Symbol
			if metric.Percent != test.want || metric.Passed != test.wantPassed {
				t.Fatalf("symbol=%+v want percent=%v passed=%t", metric, test.want, test.wantPassed)
			}
			if !test.wantPassed && metric.FailureCode != "CONCENTRATION_SYMBOL_EXCEEDED" {
				t.Fatalf("failure code=%q", metric.FailureCode)
			}
		})
	}
}

func TestCountAlternativeTemporalBoundariesAndEmptyBuckets(t *testing.T) {
	policy := GovernanceConcentrationAlternatives()[0]
	for _, test := range []struct {
		name       string
		dominant   int
		wantPassed bool
	}{
		{"below", 49, true}, {"at", 50, true}, {"above", 51, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := evaluateSyntheticAlternative(t, policy, countMonthFixture(t, test.dominant))
			if got.Partitions[0].Temporal.Percent != float64(test.dominant) || got.Partitions[0].Temporal.Passed != test.wantPassed {
				t.Fatalf("temporal=%+v", got.Partitions[0].Temporal)
			}
		})
	}

	base := time.Date(2030, 1, 31, 23, 59, 0, 0, time.UTC)
	rows := []ConcentrationObservation{
		countObservation(t, "P", singletonConcentrationCluster(t, "AAAUSDT", base)),
		countObservation(t, "P", singletonConcentrationCluster(t, "BBBUSDT", time.Date(2030, 3, 1, 0, 1, 0, 0, time.UTC))),
	}
	got := evaluateSyntheticAlternative(t, policy, rows)
	if got.Partitions[0].Temporal.Percent != 50 {
		t.Fatalf("empty February changed denominator: %+v", got.Partitions[0].Temporal)
	}
}

func TestCountAlternativeMonthAndQuarterBoundaryUTC(t *testing.T) {
	policy := GovernanceConcentrationAlternatives()[0]
	rows := []ConcentrationObservation{
		countObservation(t, "P", singletonConcentrationCluster(t, "AAAUSDT", time.Date(2030, 3, 31, 23, 59, 59, 999999999, time.UTC))),
		countObservation(t, "P", singletonConcentrationCluster(t, "BBBUSDT", time.Date(2030, 4, 1, 0, 0, 0, 0, time.UTC))),
	}
	got := evaluateSyntheticAlternative(t, policy, rows)
	if got.Partitions[0].Temporal.Percent != 50 {
		t.Fatalf("UTC month/quarter boundary=%+v", got.Partitions[0].Temporal)
	}
}

func TestCountAlternativeLargestAndAggregateClusterBoundaries(t *testing.T) {
	policy := GovernanceConcentrationAlternatives()[0]
	for _, test := range []struct {
		name       string
		sizes      []int
		metric     func(PartitionConcentrationResult) ConcentrationMetricResult
		want       float64
		wantPassed bool
	}{
		{"largest below", []int{49, 26, 25}, func(r PartitionConcentrationResult) ConcentrationMetricResult { return r.LargestCluster }, 49, true},
		{"largest at", []int{50, 25, 25}, func(r PartitionConcentrationResult) ConcentrationMetricResult { return r.LargestCluster }, 50, true},
		{"largest above", []int{51, 25, 24}, func(r PartitionConcentrationResult) ConcentrationMetricResult { return r.LargestCluster }, 51, false},
		{"top five below", []int{14, 14, 14, 14, 13, 9, 9, 9, 2, 2}, func(r PartitionConcentrationResult) ConcentrationMetricResult { return r.AggregateCluster }, 69, true},
		{"top five at", []int{14, 14, 14, 14, 14, 6, 6, 6, 6, 6}, func(r PartitionConcentrationResult) ConcentrationMetricResult { return r.AggregateCluster }, 70, true},
		{"top five above", []int{15, 14, 14, 14, 14, 9, 8, 6, 3, 3}, func(r PartitionConcentrationResult) ConcentrationMetricResult { return r.AggregateCluster }, 71, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := evaluateSyntheticAlternative(t, policy, countSizeFixture(t, test.sizes))
			metric := test.metric(got.Partitions[0])
			if metric.Percent != test.want || metric.Passed != test.wantPassed {
				t.Fatalf("metric=%+v", metric)
			}
		})
	}
}

func TestCountAlternativeMultiSymbolDuplicatesRawDifferenceAndTies(t *testing.T) {
	policy := GovernanceConcentrationAlternatives()[0]
	base := time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC)
	left := syntheticEvent(t, "AAAUSDT", base)
	right := withSyntheticEpisodeV2(t, syntheticEvent(t, "BBBUSDT", base.Add(time.Minute)), left)
	multi := mustClusterV2(t, []RetainedEvent{left, right}, RevisedIndependencePolicyV2())[0]
	rows := []ConcentrationObservation{countObservation(t, "P", multi)}
	for i, symbol := range []string{"CCCUSDT", "DDDUSDT", "EEEUSDT"} {
		rows = append(rows, countObservation(t, "P", singletonConcentrationCluster(t, symbol, base.Add(time.Duration(i+1)*24*time.Hour))))
	}
	got := evaluateSyntheticAlternative(t, policy, rows)
	if got.Partitions[0].Symbol.Percent != 25 {
		t.Fatalf("multi-symbol fractional attribution=%+v", got.Partitions[0].Symbol)
	}

	duplicate := syntheticEvent(t, "DUPUSDT", base.Add(10*24*time.Hour))
	clusters := mustClusterV2(t, []RetainedEvent{duplicate, duplicate}, RevisedIndependencePolicyV2())
	if len(clusters) != 1 || len(clusters[0].MemberEventIDs) != 1 {
		t.Fatal("canonical duplicate event inflated cluster membership")
	}

	rawDifference := []ConcentrationObservation{countObservation(t, "P", sizedConcentrationCluster(t, base.Add(20*24*time.Hour), 9, "AAAUSDT"))}
	for i := 0; i < 9; i++ {
		rawDifference = append(rawDifference, countObservation(t, "P", singletonConcentrationCluster(t, fmt.Sprintf("S%02dUSDT", i), base.Add(time.Duration(40+i)*24*time.Hour))))
	}
	rawGot := evaluateSyntheticAlternative(t, policy, rawDifference)
	rawEventShare := 9.0 / 18.0 * 100
	if rawGot.Partitions[0].Symbol.Percent != 10 || rawEventShare != 50 {
		t.Fatalf("cluster unit=%v raw unit=%v", rawGot.Partitions[0].Symbol.Percent, rawEventShare)
	}

	equal := evaluateSyntheticAlternative(t, policy, countSizeFixture(t, []int{2, 2, 1, 1, 1, 1, 1, 1}))
	if equal.Partitions[0].LargestCluster.Percent != 20 {
		t.Fatalf("equal largest clusters=%+v", equal.Partitions[0].LargestCluster)
	}
	tiedTopN := evaluateSyntheticAlternative(t, policy, countSizeFixture(t, []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}))
	if tiedTopN.Partitions[0].AggregateCluster.Percent != 50 {
		t.Fatalf("top-N ties=%+v", tiedTopN.Partitions[0].AggregateCluster)
	}
}

func TestPositiveReturnAlternativeAllMetricBoundaries(t *testing.T) {
	policy := GovernanceConcentrationAlternatives()[1]
	for _, test := range []struct {
		name       string
		values     []float64
		metric     func(PartitionConcentrationResult) ConcentrationMetricResult
		want       float64
		wantPassed bool
	}{
		{"largest below", []float64{49, 26, 25}, func(r PartitionConcentrationResult) ConcentrationMetricResult { return r.LargestCluster }, 49, true},
		{"largest at", []float64{50, 25, 25}, func(r PartitionConcentrationResult) ConcentrationMetricResult { return r.LargestCluster }, 50, true},
		{"largest above", []float64{51, 25, 24}, func(r PartitionConcentrationResult) ConcentrationMetricResult { return r.LargestCluster }, 51, false},
		{"top five below", []float64{14, 14, 14, 14, 13, 9, 9, 9, 2, 2}, func(r PartitionConcentrationResult) ConcentrationMetricResult { return r.AggregateCluster }, 69, true},
		{"top five at", []float64{14, 14, 14, 14, 14, 6, 6, 6, 6, 6}, func(r PartitionConcentrationResult) ConcentrationMetricResult { return r.AggregateCluster }, 70, true},
		{"top five above", []float64{15, 14, 14, 14, 14, 9, 8, 6, 3, 3}, func(r PartitionConcentrationResult) ConcentrationMetricResult { return r.AggregateCluster }, 71, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := evaluateSyntheticAlternative(t, policy, returnFixture(t, test.values, nil))
			metric := test.metric(got.Partitions[0])
			if metric.Percent != test.want || metric.Passed != test.wantPassed {
				t.Fatalf("metric=%+v", metric)
			}
		})
	}

	for _, test := range []struct {
		name       string
		dominant   float64
		wantPassed bool
	}{
		{"symbol below", 49, true}, {"symbol at", 50, true}, {"symbol above", 51, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			values := []float64{test.dominant, 26, 25}
			values[2] = 100 - values[0] - values[1]
			got := evaluateSyntheticAlternative(t, policy, returnFixture(t, values, []string{"AAAUSDT", "BBBUSDT", "CCCUSDT"}))
			if got.Partitions[0].Symbol.Percent != test.dominant || got.Partitions[0].Symbol.Passed != test.wantPassed {
				t.Fatalf("symbol=%+v", got.Partitions[0].Symbol)
			}
		})
	}
}

func TestPositiveReturnAlternativeTemporalMultiSymbolAndZeroDenominator(t *testing.T) {
	policy := GovernanceConcentrationAlternatives()[1]
	base := time.Date(2032, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := returnFixtureAtTimes(t, []float64{50, 50}, []string{"AAAUSDT", "BBBUSDT"}, []time.Time{base, base.AddDate(0, 1, 0)})
	got := evaluateSyntheticAlternative(t, policy, rows)
	if got.Partitions[0].Temporal.Percent != 50 || !got.Partitions[0].Temporal.Passed {
		t.Fatalf("temporal equality=%+v", got.Partitions[0].Temporal)
	}

	left := syntheticEvent(t, "AAAUSDT", base.AddDate(0, 2, 0))
	right := withSyntheticEpisodeV2(t, syntheticEvent(t, "BBBUSDT", base.AddDate(0, 2, 0).Add(time.Minute)), left)
	multi := mustClusterV2(t, []RetainedEvent{left, right}, RevisedIndependencePolicyV2())[0]
	value := 100.0
	multiGot := evaluateSyntheticAlternative(t, policy, []ConcentrationObservation{{Partition: "P", Cluster: multi, ClusterNetReturn: &value}})
	if multiGot.Partitions[0].Symbol.Percent != 50 {
		t.Fatalf("positive-return fractional multi-symbol=%+v", multiGot.Partitions[0].Symbol)
	}

	negative := -1.0
	_, err := EvaluateConcentrationAlternative(policy, RevisedIndependencePolicyV2(), []string{"P"}, []ConcentrationObservation{{Partition: "P", Cluster: singletonConcentrationCluster(t, "AAAUSDT", base), ClusterNetReturn: &negative}})
	if err == nil {
		t.Fatal("zero positive-return denominator defaulted to pass")
	}
}

func TestConcentrationAlternativesDeterminismRoundingAndFailClosed(t *testing.T) {
	for _, policy := range GovernanceConcentrationAlternatives() {
		t.Run(policy.MetricVersion, func(t *testing.T) {
			rows := countSizeFixture(t, []int{1, 1, 1})
			if policy.Basis == ConcentrationBasisPositiveClusterReturn {
				for i := range rows {
					value := 1.0
					rows[i].ClusterNetReturn = &value
				}
			}
			forward := evaluateSyntheticAlternative(t, policy, rows)
			for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
				rows[i], rows[j] = rows[j], rows[i]
			}
			reverse := evaluateSyntheticAlternative(t, policy, rows)
			left, _ := json.Marshal(forward)
			right, _ := json.Marshal(reverse)
			if string(left) != string(right) {
				t.Fatal("reordered input changed deterministic evidence")
			}
			if forward.Partitions[0].LargestCluster.Percent != 33.333333 {
				t.Fatalf("rounding=%v", forward.Partitions[0].LargestCluster.Percent)
			}

			mutated := policy
			mutated.SymbolThresholdPercent++
			if _, err := ConcentrationAlternativeHash(mutated); err == nil {
				t.Fatal("same-version policy mutation was accepted")
			}
		})
	}

	policy := GovernanceConcentrationAlternatives()[0]
	cluster := singletonConcentrationCluster(t, "AAAUSDT", time.Date(2033, 1, 1, 0, 0, 0, 0, time.UTC))
	tests := []struct {
		name   string
		mutate func(*IndependentClusterV2)
	}{
		{"malformed cluster identity", func(c *IndependentClusterV2) { c.ClusterID += "-bad" }},
		{"missing symbol", func(c *IndependentClusterV2) { c.MemberSymbols = []string{""} }},
		{"missing timestamp", func(c *IndependentClusterV2) { c.MemberEventTimes[0] = time.Time{} }},
		{"policy hash mutation", func(c *IndependentClusterV2) { c.PolicyHash = "sha256:" + fmt.Sprintf("%064d", 1) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bad := cluster
			bad.MemberSymbols = append([]string(nil), cluster.MemberSymbols...)
			bad.MemberEventTimes = append([]time.Time(nil), cluster.MemberEventTimes...)
			test.mutate(&bad)
			_, err := EvaluateConcentrationAlternative(policy, RevisedIndependencePolicyV2(), []string{"P"}, []ConcentrationObservation{countObservation(t, "P", bad)})
			if err == nil {
				t.Fatal("invalid concentration evidence passed")
			}
		})
	}

	if _, err := EvaluateConcentrationAlternative(policy, RevisedIndependencePolicyV2(), []string{"P"}, nil); err == nil {
		t.Fatal("zero denominator partition passed")
	}
	if _, err := EvaluateConcentrationAlternative(policy, RevisedIndependencePolicyV2(), []string{"P"}, []ConcentrationObservation{{Partition: "", Cluster: cluster}}); err == nil {
		t.Fatal("missing partition passed")
	}
}

func evaluateSyntheticAlternative(t *testing.T, policy ConcentrationAlternativePolicy, rows []ConcentrationObservation) ConcentrationEvaluation {
	t.Helper()
	got, err := EvaluateConcentrationAlternative(policy, RevisedIndependencePolicyV2(), []string{"P"}, rows)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func countSymbolFixture(t *testing.T, dominant int) []ConcentrationObservation {
	t.Helper()
	base := time.Date(2034, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := make([]ConcentrationObservation, 0, 100)
	remaining := 100 - dominant
	second := remaining / 2
	for i := 0; i < 100; i++ {
		symbol := "CCCUSDT"
		if i < dominant {
			symbol = "AAAUSDT"
		} else if i < dominant+second {
			symbol = "BBBUSDT"
		}
		rows = append(rows, countObservation(t, "P", singletonConcentrationCluster(t, symbol, base.Add(time.Duration(i)*5*time.Hour))))
	}
	return rows
}

func countMonthFixture(t *testing.T, dominant int) []ConcentrationObservation {
	t.Helper()
	counts := []int{dominant, (100 - dominant) / 2, 100 - dominant - (100-dominant)/2}
	rows := make([]ConcentrationObservation, 0, 100)
	for monthIndex, count := range counts {
		base := time.Date(2035, time.Month(monthIndex+1), 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < count; i++ {
			rows = append(rows, countObservation(t, "P", singletonConcentrationCluster(t, fmt.Sprintf("M%02dUSDT", monthIndex), base.Add(time.Duration(i)*5*time.Hour))))
		}
	}
	return rows
}

func countSizeFixture(t *testing.T, sizes []int) []ConcentrationObservation {
	t.Helper()
	base := time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := make([]ConcentrationObservation, 0, len(sizes))
	for i, size := range sizes {
		rows = append(rows, countObservation(t, "P", sizedConcentrationCluster(t, base.Add(time.Duration(i)*48*time.Hour), size, fmt.Sprintf("C%02dUSDT", i))))
	}
	return rows
}

func returnFixture(t *testing.T, values []float64, symbols []string) []ConcentrationObservation {
	t.Helper()
	times := make([]time.Time, len(values))
	base := time.Date(2037, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range times {
		times[i] = base.Add(time.Duration(i) * 5 * time.Hour)
	}
	return returnFixtureAtTimes(t, values, symbols, times)
}

func returnFixtureAtTimes(t *testing.T, values []float64, symbols []string, times []time.Time) []ConcentrationObservation {
	t.Helper()
	rows := make([]ConcentrationObservation, len(values))
	for i, value := range values {
		symbol := fmt.Sprintf("R%02dUSDT", i)
		if len(symbols) > i {
			symbol = symbols[i]
		}
		valueCopy := value
		rows[i] = ConcentrationObservation{Partition: "P", Cluster: singletonConcentrationCluster(t, symbol, times[i]), ClusterNetReturn: &valueCopy}
	}
	return rows
}

func singletonConcentrationCluster(t *testing.T, symbol string, at time.Time) IndependentClusterV2 {
	t.Helper()
	return mustClusterV2(t, []RetainedEvent{syntheticEvent(t, symbol, at)}, RevisedIndependencePolicyV2())[0]
}

func sizedConcentrationCluster(t *testing.T, base time.Time, size int, symbol string) IndependentClusterV2 {
	t.Helper()
	if size <= 0 {
		t.Fatal("synthetic cluster size must be positive")
	}
	events := make([]RetainedEvent, size)
	for i := range events {
		events[i] = syntheticEvent(t, symbol, base.Add(time.Duration(i)*time.Minute))
	}
	clusters := mustClusterV2(t, events, RevisedIndependencePolicyV2())
	if len(clusters) != 1 || len(clusters[0].MemberEventIDs) != size {
		t.Fatalf("synthetic sized cluster=%d members=%d want=%d", len(clusters), len(clusters[0].MemberEventIDs), size)
	}
	return clusters[0]
}

func countObservation(t *testing.T, partition string, cluster IndependentClusterV2) ConcentrationObservation {
	t.Helper()
	return ConcentrationObservation{Partition: partition, Cluster: cluster}
}

func TestRoundSixIsDeterministic(t *testing.T) {
	if got := roundSix(100.0 / 3.0); math.Abs(got-33.333333) > 1e-12 {
		t.Fatalf("roundSix=%v", got)
	}
}
