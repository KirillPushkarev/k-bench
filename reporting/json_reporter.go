package reporting

import (
	"encoding/json"
	"fmt"
	"golang.org/x/exp/maps"
	"k-bench/manager"
	"os"
)

type JsonReporter struct {
}

func (reporter JsonReporter) Report(mgrs map[string]manager.Manager) error {
	var allStats map[string]manager.Stats

	for kind, mgr := range mgrs {
		mgrStats := mgr.GetStats()
		kindStats := allStats[kind]
		kindStats.SetPodStats(mgrStats.PodStats())
		maps.Copy(kindStats.ApiCallStats(), mgrStats.ApiCallStats())
	}

	content, err := json.Marshal(map[string]map[string]manager.Stats{"metrics": allStats})
	if err != nil {
		fmt.Println(err)
	}
	err = os.WriteFile("report.json", content, 0644)

	return nil
}
