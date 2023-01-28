package reporting

import (
	"encoding/json"
	"fmt"
	"k-bench/manager"
	"k-bench/perf_util"
	"os"
)

type JsonReporter struct {
}

type StatsModel struct {
	podStats             *manager.PodStats
	apiCallStatsByMethod map[string]perf_util.OperationLatencyMetric
}

func NewStatsModel(podStats *manager.PodStats, apiCallStatsByMethod map[string]perf_util.OperationLatencyMetric) *StatsModel {
	return &StatsModel{podStats: podStats, apiCallStatsByMethod: apiCallStatsByMethod}
}

func (reporter JsonReporter) Report(mgrs map[string]manager.Manager) error {
	allStats := make(map[string]StatsModel)

	for mgrKind, mgr := range mgrs {
		mgrStats := mgr.GetStats()
		if mgrKind != "Resource" {
			allStats[mgrKind] = *NewStatsModel(mgrStats.PodStats(), mgrStats.ApiCallStats()[mgrKind])
		} else {
			for resourceKind := range mgrStats.ApiCallStats() {
				allStats[resourceKind] = *NewStatsModel(nil, mgrStats.ApiCallStats()[resourceKind])
			}
		}
	}

	content, err := json.Marshal(map[string]map[string]StatsModel{"metrics": allStats})
	if err != nil {
		fmt.Println(err)
	}
	err = os.WriteFile("report.json", content, 0644)

	return nil
}
