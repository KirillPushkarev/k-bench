package logging

import (
	log "github.com/sirupsen/logrus"
	"k-bench/perf_util"
)

func LogApiLatencies(resourceType string, apiCallLatency map[string]perf_util.OperationLatencyMetric) {
	log.Infof("--------------------------------- %v API Call Latencies (ms) "+
		"--------------------------------", resourceType)
	log.Infof("%-50v %-10v %-10v %-10v %-10v", " ", "median", "min", "max", "99%")

	for method, operationLatency := range apiCallLatency {
		if operationLatency.Valid {
			latency := operationLatency.Latency
			log.Infof("%-50v %-10v %-10v %-10v %-10v", method+" "+resourceType+" latency: ", latency.Mid, latency.Min, latency.Max, latency.P99)
		} else {
			log.Infof("%-50v %-10v %-10v %-10v %-10v", method+" "+resourceType+" latency: ", "---", "---", "---", "---")
		}
	}
}
