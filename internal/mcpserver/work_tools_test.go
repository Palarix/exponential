package mcpserver

import (
	"slices"
	"testing"

	"github.com/palarix/exponential/internal/config"
)

var workToolNames = []string{"start", "merge"}

func TestWorkTools_OnLocalStdioServer(t *testing.T) {
	_, cleanup := setup(t)
	defer cleanup()
	names := listToolNames(t, newStdioServer(&config.Config{Prefix: "test", Version: 2}))
	for _, name := range workToolNames {
		if !slices.Contains(names, name) {
			t.Errorf("local stdio server is missing %s", name)
		}
	}
}
