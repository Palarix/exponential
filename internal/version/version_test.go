package version

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.1.0", "1.0.1", 1},
		{"2.0.0", "1.9.9", 1},
		{"1.2.1", "1.2.1", 0},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.0", "v1.0.0", 0},
		{"1.10.0", "1.9.0", 1},
		{"0.8.2", "0.9.0", -1},
	}

	for _, tt := range tests {
		got := CompareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestFormatDescribe(t *testing.T) {
	tests := []struct {
		describe, want string
	}{
		{"v1.5.0", "1.5.0"},
		{"v1.5.0!", "1.5.0!"},
		{"v1.5.0-28-gde0d346", "1.5.0-de0d346"},
		{"v1.5.0-28-gde0d346!", "1.5.0-de0d346!"},
		{"v1.5.0-1-gde0d3461dbfe", "1.5.0-de0d3461dbfe"},
		{"v1.5.0-rc1", "1.5.0-rc1"},
		{"v1.5.0-rc1-3-gde0d346", "1.5.0-rc1-de0d346"},
	}

	for _, tt := range tests {
		if got := formatDescribe(tt.describe); got != tt.want {
			t.Errorf("formatDescribe(%q) = %q, want %q", tt.describe, got, tt.want)
		}
	}
}

func TestFormatBuildInfo(t *testing.T) {
	const revision = "de0d3461dbfe4c17e0563b8530ce83dc2536e540"
	const pseudo = "v1.5.1-0.20261106165833-de0d3461dbfe"

	tests := []struct {
		name                  string
		mainVersion, revision string
		modified              bool
		want                  string
	}{
		{"between tags", pseudo, revision, false, "1.5.0-de0d346"},
		{"between tags, uncommitted", pseudo + "+dirty", revision, true, "1.5.0-de0d346!"},
		{"on the tag", "v1.5.0", revision, false, "1.5.0"},
		{"on the tag, uncommitted", "v1.5.0+dirty", revision, true, "1.5.0!"},
		{"on another tag", "v1.4.0", revision, false, "1.5.0-de0d346"},
		{"no main version", "(devel)", revision, false, "1.5.0-de0d346"},
		{"no git data", "(devel)", "", false, "1.5.0"},
		{"short revision", pseudo, "de0d3", false, "1.5.0-de0d3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBuildInfo("1.5.0", tt.mainVersion, tt.revision, tt.modified)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildPrefersLinkedDescribe(t *testing.T) {
	old := gitDescribe
	t.Cleanup(func() { gitDescribe = old })

	gitDescribe = "v9.9.9-2-gabc1234!"
	if got := Build(); got != "9.9.9-abc1234!" {
		t.Errorf("Build() = %q, want the linked describe value", got)
	}
}

func TestBuildWithoutGitDataIsCLIVersion(t *testing.T) {
	// Test binaries have no linked value and no VCS data.
	if got := Build(); got != CLIVersion {
		t.Errorf("Build() = %q, want %q", got, CLIVersion)
	}
}
