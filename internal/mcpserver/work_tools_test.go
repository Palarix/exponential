package mcpserver

import (
	"slices"
	"testing"

	"github.com/palarix/exponential/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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

func TestWorkTools_NotOnRemoteStdioServer(t *testing.T) {
	_, cleanup := setup(t)
	defer cleanup()
	cfg := &config.Config{Prefix: "test", Version: 2}
	cfg.Remote.URL = "https://xpo.example.com"
	names := listToolNames(t, newStdioServer(cfg))
	for _, name := range workToolNames {
		if slices.Contains(names, name) {
			t.Errorf("stdio server in remote mode offers %s", name)
		}
	}
	if !slices.Contains(names, "show") || !slices.Contains(names, "git_diff") {
		t.Error("stdio server in remote mode lost its other tools")
	}
}

func TestWorkTools_NotOnServe(t *testing.T) {
	_, cleanup := setup(t)
	defer cleanup()
	srv := mcp.NewServer(&mcp.Implementation{Name: "xpo", Version: "v0"}, nil)
	RegisterTools(srv, &config.Config{Prefix: "test", Version: 2}, nil, nil)
	names := listToolNames(t, srv)
	for _, name := range workToolNames {
		if slices.Contains(names, name) {
			t.Errorf("xpo serve's /mcp offers %s", name)
		}
	}
}
