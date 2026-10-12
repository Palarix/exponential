package chat

import "fmt"

// AgentStatus tells the UI whether chat can run, and why not.
type AgentStatus struct {
	Available bool   `json:"available"`
	Agent     string `json:"agent"`
	Model     string `json:"model"`
	Reason    string `json:"reason,omitempty"`
}

// CheckAgent resolves the configured chat agent (chat.agent, chat.model) and
// checks that its CLI is installed. Only claude is supported for now.
func CheckAgent(agent, model string, lookPath func(string) (string, error)) AgentStatus {
	if agent == "" {
		agent = "claude"
	}
	st := AgentStatus{Agent: agent, Model: model}
	if agent != "claude" {
		st.Reason = fmt.Sprintf("Chat supports only the claude agent, not %q (chat.agent in .xpo/config.yaml).", agent)
		return st
	}
	if _, err := lookPath("claude"); err != nil {
		st.Reason = "Claude Code (claude) is not on PATH. Install it from https://code.claude.com and restart xpo board."
		return st
	}
	st.Available = true
	return st
}
