package version

import (
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
)

// CLIVersion is the release version of the xpo CLI, always plain x.y.z. It
// stamps integration files and takes part in version comparisons. To show the
// running version, use Build.
const CLIVersion = "1.5.0"

// gitDescribe holds the raw `git describe --tags --dirty='!'` output. The
// Makefile sets it with -ldflags -X.
var gitDescribe string

// dirtyMark ends the version of a build with uncommitted changes.
const dirtyMark = "!"

// shortRevisionLen matches the length of git's default short SHA.
const shortRevisionLen = 7

// describeCommitsRe matches the "-<commits>-g<sha>" that git describe adds when
// HEAD is past the tag.
var describeCommitsRe = regexp.MustCompile(`-\d+-g([0-9a-f]+)(!?)$`)

// Build returns the version of the running binary, for display: 1.5.0 on a
// tag, 1.5.0-de0d346 past it, with a trailing ! for uncommitted changes.
func Build() string {
	if gitDescribe != "" {
		return formatDescribe(gitDescribe)
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return CLIVersion
	}
	var revision string
	var modified bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	return formatBuildInfo(CLIVersion, info.Main.Version, revision, modified)
}

// formatDescribe converts git describe output to the display format. It drops
// the tag's leading v, the commit count and the g before the SHA.
func formatDescribe(describe string) string {
	return describeCommitsRe.ReplaceAllString(strings.TrimPrefix(describe, "v"), "-$1$2")
}

// formatBuildInfo builds the display format from Go's embedded VCS data, for
// builds without the Makefile's flags. mainVersion is only a check for a build
// exactly on the release tag: past a tag, Go reports a pseudo-version of the
// next patch, which is not a real version.
func formatBuildInfo(release, mainVersion, revision string, modified bool) string {
	if revision == "" {
		return release
	}
	mark := ""
	if modified {
		mark = dirtyMark
	}
	if strings.TrimSuffix(mainVersion, "+dirty") == "v"+release {
		return release + mark
	}
	if len(revision) > shortRevisionLen {
		revision = revision[:shortRevisionLen]
	}
	return release + "-" + revision + mark
}

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
