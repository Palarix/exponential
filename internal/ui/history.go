package ui

import (
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/palarix/exponential/internal/model"
)

// RenderHistory renders the event history for an issue.
func RenderHistory(events []model.Event, termWidth int) {
	if len(events) == 0 {
		fmt.Println("No events found.")
		return
	}

	// Column widths
	timeWidth := 16
	typeWidth := 12
	userWidth := 20
	detailWidth := termWidth - timeWidth - typeWidth - userWidth - 6
	if detailWidth < 10 {
		detailWidth = 10
	}

	// Header
	header := fmt.Sprintf("%s %s %s %s",
		RenderCell("TIME", timeWidth, MutedStyle),
		RenderCell("TYPE", typeWidth, MutedStyle),
		RenderCell("BY", userWidth, MutedStyle),
		RenderCell("DETAILS", detailWidth, MutedStyle),
	)
	fmt.Println(header)
	fmt.Println(MutedStyle.Render(strings.Repeat("─", termWidth)))

	for _, evt := range events {
		timeStr := humanize.Time(evt.CreatedAt)
		typeStr := string(evt.Type)
		userStr := Truncate(ExtractName(evt.CreatedBy), userWidth)

		detail := ""
		switch evt.Type {
		case model.EventTypeUpdate:
			detail = "Updated issue"
		case model.EventTypeComment:
			detail = "Added comment"
		case model.EventTypeDelete:
			detail = "Deleted issue"
		case model.EventTypeCreate:
			detail = "Created issue"
		}
		detail = Truncate(detail, detailWidth)

		row := fmt.Sprintf("%s %s %s %s",
			RenderCell(timeStr, timeWidth, MutedStyle),
			RenderCell(typeStr, typeWidth, AccentStyle),
			RenderCell(userStr, userWidth, WhiteStyle),
			RenderCell(detail, detailWidth, MutedStyle),
		)
		fmt.Println(row)
	}
}
