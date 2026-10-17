package main

import (
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/config"
)

func TestWhoamiIdentity_ConfiguredUser(t *testing.T) {
	got := whoamiIdentity(&config.Config{User: "Ada <ada@example.com>"})
	if got != "Ada <ada@example.com>" {
		t.Errorf("identity = %q, want the configured user", got)
	}
}

// Outside a project there is no config; the identity comes from git.
func TestWhoamiIdentity_NoConfig(t *testing.T) {
	if got := whoamiIdentity(nil); !strings.Contains(got, "<") {
		t.Errorf("identity = %q, want a Name <email> identity", got)
	}
}
