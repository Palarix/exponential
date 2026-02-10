package beats

import (
	"fmt"
)

// RunMigrations is a no-op for v2. The v2 data model is a breaking change.
// Users should run `beats init` to create a fresh issues.db.
func RunMigrations(currentVersion int) (int, error) {
	if currentVersion >= 2 {
		return currentVersion, nil
	}
	return 0, fmt.Errorf("data model v%d is not compatible with v2; please run `beats init` to create a fresh database", currentVersion)
}
