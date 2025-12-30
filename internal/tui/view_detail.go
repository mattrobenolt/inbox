package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) renderDetailView() string {
	if m.detail.loading {
		footer := m.renderDetailStatusline(0, false, true, m.detail.messageViewMode)
		return renderFixedLayout(m.ui.height, "", footer)
	}

	if len(m.detail.messages) == 0 {
		footer := m.renderDetailStatusline(0, false, false, m.detail.messageViewMode)
		return renderFixedLayout(m.ui.height, "No messages loaded", footer)
	}

	// Scrollable body with all messages (viewport renders its own content with padding from cards)
	body := m.detail.viewport.View()
	scrollPercent := int(m.detail.viewport.ScrollPercent() * 100)

	canToggle := false
	selectedMode := m.detail.messageViewMode
	if m.detail.selectedMessageIdx >= 0 && m.detail.selectedMessageIdx < len(m.detail.messages) {
		msg := m.detail.messages[m.detail.selectedMessageIdx]
		modes := availableMessageViewModes(msg)
		if m.detail.expandedMessages[msg.ID] && len(modes) > 0 {
			canToggle = true
		}
		selectedMode = normalizeMessageViewMode(selectedMode, msg)
	}
	footer := m.renderDetailStatusline(scrollPercent, canToggle, false, selectedMode)

	return renderFixedLayout(m.ui.height, body, footer)
}

func (m *Model) renderDetailStatusline(
	scrollPercent int,
	canToggle bool,
	loading bool,
	selectedMode messageViewMode,
) string {
	label := "MESSAGE"
	if len(m.detail.messages) > 1 {
		label = "THREAD"
	}
	left := []statusSegment{
		statusModeSegment(m.theme, "INBOX"),
		statusPowerlineSeparator(m.theme.Status.ModeBg, m.theme.Status.TabBg),
		statusTabSegment(m.theme, label),
		statusPowerlineSeparator(m.theme.Status.TabBg, m.theme.Status.Bg),
	}
	if len(m.detail.messages) > 1 {
		left = append(left, statusTextSegment(m.theme, fmt.Sprintf(
			"msg %d/%d",
			min(m.detail.selectedMessageIdx+1, len(m.detail.messages)),
			len(m.detail.messages),
		)))
	}

	right := []statusSegment{}
	if loading {
		right = append(right, statusDimSegment(m.theme, m.ui.spinner.View()+" loading"))
	} else {
		if canToggle {
			mode := "TEXT"
			switch selectedMode {
			case viewModeText:
				mode = "TEXT"
			case viewModeHTML:
				mode = "HTML"
			case viewModeRaw:
				mode = "RAW"
			}
			modeStyle := lipgloss.NewStyle().
				Background(lipgloss.Color(m.theme.Detail.ViewModeBg)).
				Foreground(lipgloss.Color(m.theme.Detail.ViewModeFg)).
				Bold(true).
				Padding(0, 1)
			right = append(right, statusSegment{text: mode, style: modeStyle})
		}
		right = append(right, statusTextSegment(m.theme, fmt.Sprintf("%d%%", scrollPercent)))
	}
	right = append(
		right,
		statusDimSegment(m.theme, "esc back"),
		statusDimSegment(m.theme, "? help"),
	)

	subject := ""
	if m.detail.currentThread != nil {
		subject = strings.TrimSpace(m.detail.currentThread.Subject)
	}
	if subject != "" {
		maxWidth := 0
		if m.ui.width > 0 {
			maxWidth = m.ui.width -
				lipgloss.Width(renderStatusSegments(left)) -
				lipgloss.Width(renderStatusSegments(right)) - 1
		}
		if maxWidth > 0 {
			subject = truncateToWidth(subject, maxWidth)
		}
		if subject != "" {
			left = append(left, statusDimSegment(m.theme, subject))
		}
	}

	return renderStatusline(m.theme, m.ui.width, left, right)
}

func (m *Model) renderThreadBody() string {
	var b strings.Builder

	snippetStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Detail.SnippetFg))

	metaStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Detail.HeaderLabelFg))

	selectedBarStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Status.ModeBg))

	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Status.Dim))

	selectedPrefix := selectedBarStyle.Render("┃") + " "
	normalPrefix := "  "
	contentWidth := max(m.ui.width-2, 0)

	writeLines := func(prefix string, lines []string) {
		for i, line := range lines {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(prefix)
			b.WriteString(line)
		}
	}

	for i, msg := range m.detail.messages {
		isExpanded := m.detail.expandedMessages[msg.ID]
		isSelected := i == m.detail.selectedMessageIdx
		prefix := normalPrefix
		if isSelected {
			prefix = selectedPrefix
		}

		// Extract just the name from "Name <email>"
		from := msg.From
		if idx := strings.Index(from, "<"); idx > 0 {
			from = strings.TrimSpace(from[:idx])
		}

		date := formatRelativeTime(msg.Date)

		var content strings.Builder

		if isExpanded {
			effectiveMode := normalizeMessageViewMode(m.detail.messageViewMode, msg)
			if effectiveMode == viewModeRaw {
				rawText := msg.Raw
				if rawText == "" {
					if m.detail.rawLoading[msg.ID] {
						rawText = "[Loading raw message...]"
					} else {
						rawText = "[Raw message unavailable]"
					}
				}
				content.WriteString(normalizeRawForDisplay(rawText))
			} else {
				// Expanded view - show full message with headers
				headerLabelStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color(m.theme.Detail.HeaderLabelFg)).
					Bold(true)

				headerValueStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color(m.theme.Detail.HeaderValueFg))

				content.WriteString(headerLabelStyle.Render("From: "))
				content.WriteString(headerValueStyle.Render(msg.From))
				content.WriteString("\n")
				content.WriteString(headerLabelStyle.Render("To:   "))
				content.WriteString(headerValueStyle.Render(msg.To))
				content.WriteString("\n")
				if msg.Cc != "" {
					content.WriteString(headerLabelStyle.Render("Cc:   "))
					content.WriteString(headerValueStyle.Render(msg.Cc))
					content.WriteString("\n")
				}
				content.WriteString(headerLabelStyle.Render("Date: "))
				content.WriteString(
					headerValueStyle.Render(msg.Date.Format("Mon, Jan 2, 2006 at 3:04 PM")),
				)
				content.WriteString("\n")

				// Show attachment count if any
				if len(msg.Attachments) > 0 {
					attachmentText := fmt.Sprintf("%d attachment", len(msg.Attachments))
					if len(msg.Attachments) > 1 {
						attachmentText += "s"
					}
					content.WriteString(headerLabelStyle.Render("Attachments: "))
					content.WriteString(headerValueStyle.Render(attachmentText))
					content.WriteString("\n")
				}

				content.WriteString("\n")

				// Render body based on view mode
				var bodyText string
				switch {
				case effectiveMode == viewModeHTML && msg.BodyHTML != "":
					// Clean and convert HTML to markdown, then render with glamour
					cleanedHTML := cleanHTMLForConversion(msg.BodyHTML)
					markdown, err := m.renderers.htmlConverter.ConvertString(cleanedHTML)
					if err != nil {
						// Fallback to raw HTML if conversion fails
						bodyText = msg.BodyHTML
					} else {
						bodyText = m.renderMarkdown(markdown)
					}
				case effectiveMode == viewModeText && msg.BodyText != "":
					// Show plain text through glamour
					bodyText = m.renderMarkdown(msg.BodyText)
				case msg.BodyHTML != "":
					// Fallback: clean and convert HTML to markdown if no plain text
					cleanedHTML := cleanHTMLForConversion(msg.BodyHTML)
					markdown, err := m.renderers.htmlConverter.ConvertString(cleanedHTML)
					if err != nil {
						bodyText = msg.BodyHTML
					} else {
						bodyText = m.renderMarkdown(markdown)
					}
				default:
					bodyText = lipgloss.NewStyle().Italic(true).Render("[No message body]")
				}
				content.WriteString(strings.TrimSpace(bodyText))
			}
		} else {
			// Collapsed view - show preview
			// Get first line of body as preview
			preview := strings.TrimSpace(msg.BodyText)
			if idx := strings.Index(preview, "\n"); idx > 0 {
				preview = preview[:idx]
			}
			if contentWidth > 0 {
				preview = truncateToWidth(preview, contentWidth-2)
			}

			content.WriteString(from)
			content.WriteString(" ")
			content.WriteString(metaStyle.Render("• " + date))
			content.WriteString("\n")
			content.WriteString(snippetStyle.Render(preview))
		}

		lines := strings.Split(content.String(), "\n")
		writeLines(prefix, lines)

		if i < len(m.detail.messages)-1 {
			b.WriteString("\n")
			if contentWidth > 0 {
				divider := dividerStyle.Render(strings.Repeat("─", contentWidth))
				b.WriteString(normalPrefix)
				b.WriteString(divider)
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}
