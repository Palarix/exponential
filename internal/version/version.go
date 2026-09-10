package version

import (
	"strconv"
	"strings"
)

// CLIVersion is the current version of the xpo CLI
const CLIVersion = "1.2.1"

// DataModelVersion is the current version of the data model / config schema
// v2: Simplified data model — single Issue type, labels, assignee, estimation systems, automations
// v3: Globally consistent sort_order keys across status groups
const DataModelVersion = 3

// CompareVersions compares two semver-like version strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Falls back to string comparison for non-semver strings.
func CompareVersions(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < len(aParts) || i < len(bParts); i++ {
		var aNum, bNum int
		if i < len(aParts) {
			aNum, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bNum, _ = strconv.Atoi(bParts[i])
		}
		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
	}
	return 0
}
