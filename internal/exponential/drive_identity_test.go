package exponential

import "testing"

func TestSetDriveIdentity_ActsForUser(t *testing.T) {
	c := setupClient(t)
	c.UserOverride = ""
	c.Config.User = "Nicolas <nic@example.com>"

	c.setDriveIdentity("claude", "laptop")

	if c.UserOverride != "claude <agent@laptop>" {
		t.Errorf("UserOverride = %q", c.UserOverride)
	}
	if c.OnBehalfOf != "Nicolas <nic@example.com>" {
		t.Errorf("OnBehalfOf = %q, want the configured user", c.OnBehalfOf)
	}
}
