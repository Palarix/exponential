package chat

import "errors"

// ErrCannotEdit is returned by an edit when the chat has no user message to
// edit, or the agent's session lacks the point to rewind to.
var ErrCannotEdit = errors.New("the last message cannot be edited")

// editPlan says how an edit rewinds a chat.
type editPlan struct {
	FromSeq int    // the edited user_message: it and every later event are dropped
	Anchor  string // SessionInput.Rewind
	Fresh   bool   // Claude: the edited turn was the first; start a new session
}

// activeEvents returns events without those that a rewound event dropped.
func activeEvents(events []Event) []Event {
	var out []Event
	for _, e := range events {
		if e.Type == EventRewound {
			for len(out) > 0 && out[len(out)-1].Seq >= e.FromSeq {
				out = out[:len(out)-1]
			}
			continue
		}
		out = append(out, e)
	}
	return out
}

func startsATurn(e Event) bool {
	switch e.Type {
	case EventUserMessage, EventAgentTurn, EventTurnContinued:
		return true
	}
	return false
}

func endsATurn(e Event) bool {
	switch e.Type {
	case EventTurnDone, EventTurnStopped, EventError:
		return true
	}
	return false
}

// editPoint finds the last user message of a chat and where the provider's
// session rewinds to for an edit of it. Claude resumes at the anchor of the
// turn before the message; Codex reverts the message's own turn.
func editPoint(events []Event, provider string) (editPlan, error) {
	ev := activeEvents(events)
	u := -1
	for i := len(ev) - 1; i >= 0; i-- {
		if ev[i].Type == EventUserMessage {
			u = i
			break
		}
	}
	if u < 0 {
		return editPlan{}, ErrCannotEdit
	}
	plan := editPlan{FromSeq: ev[u].Seq}
	if provider == ProviderCodex {
		for _, e := range ev[u+1:] {
			if endsATurn(e) {
				if e.Anchor == "" {
					return editPlan{}, ErrCannotEdit
				}
				plan.Anchor = e.Anchor
				return plan, nil
			}
		}
		return editPlan{}, ErrCannotEdit
	}
	for i := u - 1; i >= 0; i-- {
		switch {
		case endsATurn(ev[i]):
			if ev[i].Anchor == "" {
				return editPlan{}, ErrCannotEdit
			}
			plan.Anchor = ev[i].Anchor
			return plan, nil
		case startsATurn(ev[i]):
			// The turn before never ended: the board stopped during it.
			return editPlan{}, ErrCannotEdit
		}
	}
	plan.Fresh = true
	return plan, nil
}
