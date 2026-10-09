package main

import (
	"testing"

	"github.com/palarix/exponential/internal/model"
)

func TestCommentAuthor(t *testing.T) {
	agent := model.Comment{CreatedBy: "claude-code/2.1.263 <agent@mcp>", OnBehalfOf: "Nicolas <nic@x.com>"}
	if got := commentAuthor(agent); got != "Nicolas (via Claude Code)" {
		t.Errorf("agent comment author = %q", got)
	}
	human := model.Comment{CreatedBy: "Alice <alice@x.com>"}
	if got := commentAuthor(human); got != "Alice" {
		t.Errorf("human comment author = %q", got)
	}
}
