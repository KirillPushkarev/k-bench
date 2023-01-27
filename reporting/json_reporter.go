package reporting

import (
	"encoding/json"
	"fmt"
	"k-bench/manager"
	"os"
)

type JsonReporter struct {
}

func (reporter JsonReporter) Report(mgrs map[string]manager.Manager) error {
	var allStats []manager.Stats

	for _, mgr := range mgrs {
		stats := mgr.GetStats()
		allStats = append(allStats, stats)
	}

	content, err := json.Marshal(allStats)
	if err != nil {
		fmt.Println(err)
	}
	err = os.WriteFile("report.json", content, 0644)

	return nil
}
