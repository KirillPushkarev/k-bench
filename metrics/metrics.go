package metrics

import (
	"k-bench/common"
	"k-bench/perf_util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
	"time"
)

func CalculateLatenciesBetweenStages(firstStageTimes map[string]metav1.Time, secondStageTimes map[string]metav1.Time) []time.Duration {
	latencies := make([]time.Duration, 0)
	for podName, firstStageTime := range firstStageTimes {
		if secondStageTime, ok := secondStageTimes[podName]; ok {
			latencies = append(latencies, secondStageTime.Time.Sub(firstStageTime.Time))
		}
	}
	return latencies
}

func RoundToMicroSeconds(durations []time.Duration) []time.Duration {
	return common.MapArray(durations, func(t time.Duration) time.Duration {
		return t.Round(time.Microsecond)
	})
}

func ConvertToMilliSeconds(durations []time.Duration) []float32 {
	return common.MapArray(durations, func(t time.Duration) float32 {
		return float32(t) / float32(time.Millisecond)
	})
}

func SortDurations(durations []time.Duration) {
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
}

func CalculateDurationStatistics(durations []time.Duration) perf_util.OperationLatencyMetric {
	if len(durations) > 0 {
		var mid, min, max, p99 float32
		mid = float32(durations[len(durations)/2]) / float32(time.Millisecond)
		min = float32(durations[0]) / float32(time.Millisecond)
		max = float32(durations[len(durations)-1]) / float32(time.Millisecond)
		p99 = float32(durations[len(durations)-1-len(durations)/100]) /
			float32(time.Millisecond)

		return perf_util.OperationLatencyMetric{
			Valid:   true,
			Latency: perf_util.LatencyMetric{Mid: mid, Min: min, Max: max, P99: p99},
		}
	}

	return perf_util.OperationLatencyMetric{Valid: false}
}
