package reporting

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"k-bench/manager"
	"os"
	"path/filepath"
)

type StatsModel struct {
	PodStats              *manager.PodStats
	ApiTimesStatsByMethod map[string][]float32
}

func NewStatsModel(podStats *manager.PodStats, apiTimesStatsByMethod map[string][]float32) *StatsModel {
	return &StatsModel{PodStats: podStats, ApiTimesStatsByMethod: apiTimesStatsByMethod}
}

type JsonReporter struct {
	outDirPath string
}

func NewJsonReporter(outDirPath string) *JsonReporter {
	return &JsonReporter{outDirPath: outDirPath}
}

func (reporter JsonReporter) Report(mgrs map[string]manager.Manager) error {
	allStats := make(map[string]StatsModel)

	for mgrKind, mgr := range mgrs {
		mgrStats := mgr.GetStats()
		if mgrKind != "Resource" {
			allStats[mgrKind] = *NewStatsModel(mgrStats.PodStats, mgrStats.ApiTimesStats[mgrKind])
		} else {
			for resourceKind := range mgrStats.ApiTimesStats {
				allStats[resourceKind] = *NewStatsModel(nil, mgrStats.ApiTimesStats[resourceKind])
			}
		}
	}

	content, err := json.Marshal(map[string]map[string]StatsModel{"metrics": allStats})
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(reporter.outDirPath, "report.json"), content, 0644)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Metrics saved to file: " + filepath.Join(reporter.outDirPath, "report.json"))

	return nil
}
