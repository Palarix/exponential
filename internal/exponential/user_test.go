package exponential

import (
	"testing"

	"github.com/palarix/exponential/internal/config"
)

func TestGetUser_Override(t *testing.T) {
	tr := &LocalTransport{
		Config:       &config.Config{},
		UserOverride: "Agent <agent@test.com>",
	}
	if got := tr.GetUser(); got != "Agent <agent@test.com>" {
		t.Errorf("GetUser() = %q, want override", got)
	}
}

func TestGetUser_ConfigUser(t *testing.T) {
	tr := &LocalTransport{
		Config: &config.Config{User: "Config User <config@test.com>"},
	}
	if got := tr.GetUser(); got != "Config User <config@test.com>" {
		t.Errorf("GetUser() = %q, want config user", got)
	}
}

func TestGetUser_OverrideTakesPrecedence(t *testing.T) {
	tr := &LocalTransport{
		Config:       &config.Config{User: "Config <config@test.com>"},
		UserOverride: "Override <override@test.com>",
	}
	if got := tr.GetUser(); got != "Override <override@test.com>" {
		t.Errorf("GetUser() = %q, want override over config", got)
	}
}

func TestGetUser_FallsBackToGit(t *testing.T) {
	tr := &LocalTransport{Config: &config.Config{}}
	got := tr.GetUser()
	if got == "" {
		t.Error("GetUser() should return something even with no config")
	}
}
