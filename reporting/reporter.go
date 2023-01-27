package reporting

import "k-bench/manager"

type Reporter interface {
	Report(mgrs map[string]manager.Manager) error
}
