// Package identity interprets "Name <email>" actor strings: which are agents,
// what to call them, and who an assignment belongs to.
//
// Assignees are principal-first: an issue is always assigned to the person
// accountable for it. When an agent does the work, the agent is recorded
// as "via" rather than as the assignee.
package identity

import "strings"

// knownAgents maps MCP clientInfo names to display names.
var knownAgents = map[string]string{
	"claude-code":      "Claude Code",
	"codex-mcp-client": "Codex",
	"claude":           "Claude Code", // xpo drive supervisor
	"codex":            "Codex",
}

// Email returns the lowercased address inside angle brackets, or "".
func Email(id string) string {
	open := strings.LastIndexByte(id, '<')
	end := strings.LastIndexByte(id, '>')
	if open == -1 || end <= open {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(id[open+1 : end]))
}

// Name returns the display portion before " <", or the whole string.
func Name(id string) string {
	id = strings.TrimSpace(id)
	if i := strings.LastIndex(id, " <"); i != -1 {
		return strings.TrimSpace(id[:i])
	}
	return id
}

// IsAgent reports whether id is an agent identity: its email's local part
// is "agent" (agent@mcp, agent@<host>.local, ...).
func IsAgent(id string) bool {
	return strings.HasPrefix(Email(id), "agent@")
}

// AgentName returns a human display name for an agent identity, dropping
// any "/version" suffix: "claude-code/2.1.263 <agent@mcp>" → "Claude Code".
func AgentName(id string) string {
	name := Name(id)
	if i := strings.IndexByte(name, '/'); i != -1 {
		name = name[:i]
	}
	if known, ok := knownAgents[name]; ok {
		return known
	}
	return name
}

// Same reports whether two identities denote the same actor: by email when
// both have one, otherwise by case-insensitive comparison.
func Same(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if ea, eb := Email(a), Email(b); ea != "" && eb != "" {
		return ea == eb
	}
	return strings.EqualFold(a, b)
}

// Actor returns a principal-first display name for an event author: when an
// agent acted on someone's behalf, the principal leads and the agent follows
// as "(via …)".
func Actor(createdBy, onBehalfOf string) string {
	if onBehalfOf != "" {
		return Name(onBehalfOf) + " (via " + AgentName(createdBy) + ")"
	}
	return Name(createdBy)
}

// ResolveAssignment maps the raw assignee written by an event to the
// principal it belongs to, plus the agent (if any) doing the work for them.
func ResolveAssignment(raw, createdBy, onBehalfOf string) (assignee, via string) {
	if raw == "" {
		return "", ""
	}
	if IsAgent(raw) {
		switch {
		case onBehalfOf != "":
			return onBehalfOf, raw
		case createdBy != "" && !IsAgent(createdBy):
			return createdBy, raw
		default:
			return raw, raw
		}
	}
	// An actor other than the principal assigned the principal: the actor
	// is working for them. Assigning anyone else is a plain assignment.
	if onBehalfOf != "" && Same(raw, onBehalfOf) && !Same(createdBy, onBehalfOf) {
		return raw, createdBy
	}
	return raw, ""
}
