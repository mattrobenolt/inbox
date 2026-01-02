package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"go.withmatt.com/inbox/internal/gmail"
	"go.withmatt.com/inbox/internal/links"
)

func (m *Model) scanMessageLinksCmd(msg gmail.Message) tea.Cmd {
	if m.detail.linkScanAttempted[msg.ID] {
		return nil
	}
	m.detail.linkScanAttempted[msg.ID] = true

	return func() tea.Msg {
		body := m.messageBodyForScan(msg)
		if body == "" {
			return nil
		}
		mode := links.ScanModeKnown
		if m.linkAutoScan {
			mode = links.ScanModeLearn
		}
		m.linkResolver.ScanText(m.ctx, body, mode)
		return linkScanFinishedMsg{messageID: msg.ID}
	}
}

func (m *Model) messageBodyForScan(msg gmail.Message) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(msg.BodyText) != "" {
		parts = append(parts, msg.BodyText)
	}
	if strings.TrimSpace(msg.BodyHTML) != "" {
		cleanedHTML := cleanHTMLForConversion(msg.BodyHTML)
		if m.renderers.htmlConverter != nil {
			if markdown, err := m.renderers.htmlConverter.ConvertString(cleanedHTML); err == nil &&
				strings.TrimSpace(markdown) != "" {
				parts = append(parts, markdown)
			} else {
				parts = append(parts, cleanedHTML)
			}
		} else {
			parts = append(parts, cleanedHTML)
		}
	}
	return strings.Join(parts, "\n")
}
