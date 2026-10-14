package chat

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/palarix/exponential/internal/exponential"
)

// Approval decisions. The user picks one of the first three on a card;
// DecisionCancelled resolves the cards of a cancelled or finished turn.
const (
	DecisionAllow        = "allow"
	DecisionAllowSession = "allow_session" // allow, and allow matching calls for the rest of the chat
	DecisionDeny         = "deny"
	DecisionCancelled    = "cancelled"
)

// ErrApprovalNotFound is returned for a decision on a card that is not open.
var ErrApprovalNotFound = errors.New("no open approval request with this ID")

// ErrBadDecision is returned for a decision other than allow, allow_session or deny.
var ErrBadDecision = errors.New(`decision must be "allow", "allow_session" or "deny"`)

// ApprovalRequest is an agent asking before a tool call runs.
type ApprovalRequest struct {
	Tool   string // the tool name as the work list knows it, such as "Bash" or "mcp__xpo__merge"
	Detail string // the short argument shown next to the tool name
	Input  string // the full tool input as JSON
	ToolID string // the tool call the request belongs to; empty when the agent does not say
	// Rule is what "Allow for this chat" grants: a Claude Code permission
	// rule such as "Bash(echo hi *)", or the tool name.
	Rule string
}

// Approver asks the user and blocks until they decide. It returns one of the
// Decision values. When ctx ends first it returns DecisionCancelled.
type Approver func(ctx context.Context, req ApprovalRequest) string

// approve calls in.Approve, or denies when the session has no approver.
func (in SessionInput) approve(ctx context.Context, req ApprovalRequest) string {
	if in.Approve == nil {
		return DecisionDeny
	}
	return in.Approve(ctx, req)
}

// claudeControlRequest is a control_request line from `claude
// --permission-prompt-tool stdio`.
type claudeControlRequest struct {
	RequestID string `json:"request_id"`
	Request   struct {
		Subtype     string                       `json:"subtype"`
		ToolName    string                       `json:"tool_name"`
		Input       json.RawMessage              `json:"input"`
		Suggestions []claudePermissionSuggestion `json:"permission_suggestions"`
		ToolUseID   string                       `json:"tool_use_id"`
	} `json:"request"`
}

// claudePermissionSuggestion is a permission update Claude Code proposes for
// "always allow". Unknown fields are kept, so a suggestion goes back as sent
// apart from its destination.
type claudePermissionSuggestion map[string]any

// claudeApproval turns a can_use_tool request into an ApprovalRequest.
func claudeApproval(cr claudeControlRequest) ApprovalRequest {
	var input map[string]any
	_ = json.Unmarshal(cr.Request.Input, &input)
	return ApprovalRequest{
		Tool:   cr.Request.ToolName,
		Detail: exponential.ToolDetail(cr.Request.ToolName, input),
		Input:  string(cr.Request.Input),
		ToolID: cr.Request.ToolUseID,
		Rule:   claudeGrantRule(cr),
	}
}

// claudeGrantRule is the first suggested allow rule as "Tool(content)", or
// the tool name when Claude Code suggests none (as for MCP tools).
func claudeGrantRule(cr claudeControlRequest) string {
	for _, s := range cr.Request.Suggestions {
		if s["type"] != "addRules" || s["behavior"] != "allow" {
			continue
		}
		rules, _ := s["rules"].([]any)
		for _, r := range rules {
			rule, _ := r.(map[string]any)
			tool, _ := rule["toolName"].(string)
			if tool == "" {
				continue
			}
			if content, _ := rule["ruleContent"].(string); content != "" {
				return tool + "(" + content + ")"
			}
			return tool
		}
	}
	return cr.Request.ToolName
}

// claudePermissionResponse is the control_response line for a decision.
func claudePermissionResponse(cr claudeControlRequest, decision string) []byte {
	var resp map[string]any
	switch decision {
	case DecisionAllow, DecisionAllowSession:
		input := cr.Request.Input
		if len(input) == 0 {
			input = json.RawMessage("{}")
		}
		resp = map[string]any{"behavior": "allow", "updatedInput": input}
		if decision == DecisionAllowSession {
			resp["updatedPermissions"] = claudeSessionPermissions(cr)
		}
	case DecisionCancelled:
		resp = map[string]any{"behavior": "deny", "message": "The user cancelled the turn."}
	default:
		resp = map[string]any{"behavior": "deny", "message": "The user declined this tool call."}
	}
	return claudeControlResponse(cr.RequestID, resp)
}

// claudeSessionPermissions are Claude Code's suggestions with every
// destination set to "session". Claude Code suggests "localSettings", which
// would write the rule into the user's .claude/settings.local.json.
func claudeSessionPermissions(cr claudeControlRequest) []map[string]any {
	var out []map[string]any
	for _, s := range cr.Request.Suggestions {
		update := map[string]any{}
		for k, v := range s {
			update[k] = v
		}
		update["destination"] = "session"
		out = append(out, update)
	}
	if len(out) == 0 {
		out = append(out, map[string]any{
			"type":        "addRules",
			"rules":       []map[string]string{{"toolName": cr.Request.ToolName}},
			"behavior":    "allow",
			"destination": "session",
		})
	}
	return out
}

func claudeControlResponse(requestID string, response map[string]any) []byte {
	data, _ := json.Marshal(map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"subtype":    "success",
			"request_id": requestID,
			"response":   response,
		},
	})
	return data
}

// claudeControlError is the control_response line for a request xpo does
// not handle.
func claudeControlError(requestID, msg string) []byte {
	data, _ := json.Marshal(map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"subtype":    "error",
			"request_id": requestID,
			"error":      msg,
		},
	})
	return data
}

// codexDecision maps a decision to a Codex command or file-change approval
// decision. offered is the request's availableDecisions, empty when Codex
// does not list them. "acceptForSession" is used only when offered. A denial
// is "decline", which lets the turn go on, or "cancel" when Codex does not
// offer "decline".
func codexDecision(decision string, offered []any) string {
	has := func(d string) bool {
		for _, o := range offered {
			if o == d {
				return true
			}
		}
		return false
	}
	switch decision {
	case DecisionAllow:
		return "accept"
	case DecisionAllowSession:
		if has("acceptForSession") {
			return "acceptForSession"
		}
		return "accept"
	default:
		if len(offered) > 0 && !has("decline") {
			return "cancel"
		}
		return "decline"
	}
}
