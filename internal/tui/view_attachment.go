package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) renderAttachmentView() string {
	scrollPercent := int(m.detail.viewport.ScrollPercent() * 100)
	left := []statusSegment{statusModeSegment(m.theme, "ATTACH")}
	right := []statusSegment{
		statusTextSegment(m.theme, fmt.Sprintf("%d%%", scrollPercent)),
		statusDimSegment(m.theme, "up/down scroll"),
		statusDimSegment(m.theme, "esc back"),
	}

	infoParts := make([]string, 0, 3)
	if m.attachments.preview.filename != "" {
		infoParts = append(infoParts, m.attachments.preview.filename)
	}
	if m.attachments.preview.mimeType != "" {
		infoParts = append(infoParts, m.attachments.preview.mimeType)
	}
	if m.attachments.preview.size > 0 {
		infoParts = append(infoParts, formatAttachmentSize(m.attachments.preview.size))
	}
	info := strings.Join(infoParts, " • ")
	if info != "" {
		maxWidth := 0
		if m.ui.width > 0 {
			maxWidth = m.ui.width -
				lipgloss.Width(renderStatusSegments(left)) -
				lipgloss.Width(renderStatusSegments(right)) - 1
		}
		if maxWidth > 0 {
			info = truncateToWidth(info, maxWidth)
		}
		if info != "" {
			left = append(left, statusDimSegment(m.theme, info))
		}
	}

	footer := renderStatusline(m.theme, m.ui.width, left, right)

	body := m.detail.viewport.View()
	return renderFixedLayout(m.ui.height, body, footer)
}
