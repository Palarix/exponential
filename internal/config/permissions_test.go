package config

import (
	"testing"
)

func testPerms() PermissionsConfig {
	return PermissionsConfig{
		Roles: map[string][]string{
			"admin":     {"*"},
			"developer": {"issue.read", "issue.create", "issue.update", "issue.comment", "issue.start", "issue.merge", "issue.link"},
			"viewer":    {"issue.read", "issue.comment"},
		},
		Users: map[string]string{
			"alice@team.dev":  "admin",
			"bob@team.dev":    "developer",
			"intern@team.dev": "viewer",
		},
	}
}

func TestPermissions_Enabled(t *testing.T) {
	if !(testPerms().Enabled()) {
		t.Error("expected Enabled() = true for configured perms")
	}
	if (PermissionsConfig{}).Enabled() {
		t.Error("expected Enabled() = false for empty perms")
	}
}

func TestPermissions_RoleFor(t *testing.T) {
	p := testPerms()
	if got := p.RoleFor("alice@team.dev"); got != "admin" {
		t.Errorf("RoleFor(alice) = %q, want admin", got)
	}
	if got := p.RoleFor("unknown@team.dev"); got != "" {
		t.Errorf("RoleFor(unknown) = %q, want empty", got)
	}
}

func TestPermissions_Capabilities_Wildcard(t *testing.T) {
	p := testPerms()
	caps := p.Capabilities("admin")
	if len(caps) != len(AllCapabilities) {
		t.Errorf("admin caps = %d, want %d (all)", len(caps), len(AllCapabilities))
	}
}

func TestPermissions_Capabilities_Explicit(t *testing.T) {
	p := testPerms()
	caps := p.Capabilities("viewer")
	if len(caps) != 2 {
		t.Fatalf("viewer caps = %d, want 2", len(caps))
	}
}

func TestPermissions_HasCapability(t *testing.T) {
	p := testPerms()

	cases := []struct {
		email, cap string
		want       bool
	}{
		{"alice@team.dev", "issue.delete", true},
		{"alice@team.dev", "issue.read", true},
		{"bob@team.dev", "issue.merge", true},
		{"bob@team.dev", "issue.delete", false},
		{"intern@team.dev", "issue.read", true},
		{"intern@team.dev", "issue.create", false},
		{"unknown@team.dev", "issue.read", false},
	}
	for _, tc := range cases {
		if got := p.HasCapability(tc.email, tc.cap); got != tc.want {
			t.Errorf("HasCapability(%q, %q) = %v, want %v", tc.email, tc.cap, got, tc.want)
		}
	}
}

func TestPermissions_HasCapability_NoneConfigured(t *testing.T) {
	p := PermissionsConfig{}
	if !p.HasCapability("anyone@example.com", "issue.delete") {
		t.Error("expected open access when no permissions configured")
	}
}

func TestPermissions_UnknownRole(t *testing.T) {
	p := testPerms()
	if caps := p.Capabilities("nonexistent"); caps != nil {
		t.Errorf("expected nil caps for unknown role, got %v", caps)
	}
}
