package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type rightPart struct {
	text  string
	style lipgloss.Style
}

func rightPartsWidth(parts []rightPart) int {
	width := 0
	count := 0
	for _, part := range parts {
		if part.text == "" {
			continue
		}
		if count > 0 {
			width++
		}
		width += lipgloss.Width(part.text)
		count++
	}
	if count > 0 {
		width++
	}
	return width
}

func rightPartsText(parts []rightPart) string {
	var b strings.Builder
	first := true
	for _, part := range parts {
		if part.text == "" {
			continue
		}
		if !first {
			b.WriteString(" ")
		}
		b.WriteString(part.text)
		first = false
	}
	if b.Len() == 0 {
		return ""
	}
	return " " + b.String()
}

func renderRightParts(parts []rightPart) string {
	var b strings.Builder
	first := true
	for _, part := range parts {
		if part.text == "" {
			continue
		}
		if !first {
			b.WriteString(" ")
		}
		b.WriteString(part.style.Render(part.text))
		first = false
	}
	if b.Len() == 0 {
		return ""
	}
	return " " + b.String()
}

// getVisibleThreadRange calculates which threads should be visible.
func (m *Model) getVisibleThreadRange() (start, end int) {
	total := m.displayCount()
	if total == 0 {
		return 0, 0
	}

	visibleCards := m.visibleCardCount()
	if visibleCards <= 0 || total <= visibleCards {
		return 0, total
	}

	maxStart := max(total-visibleCards, 0)
	start = min(max(m.inbox.scrollOffset, 0), maxStart)
	end = start + visibleCards
	return start, end
}

func (m *Model) renderListView() string {
	var body strings.Builder

	// Styles
	unreadStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.List.UnreadFg)).
		Bold(true)

	readStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.List.ReadFg))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Status.Dim))

	snippetStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Status.Dim)).
		Faint(true)

	unreadSnippetStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Status.Dim))

	snippetLines := m.uiConfig.ListSnippetLines
	if snippetLines <= 0 {
		snippetLines = 1
	}

	cardPadding := 0
	cardBodyHeight := snippetLines + 2
	cardStyle := lipgloss.NewStyle().
		Padding(0, cardPadding).
		Height(cardBodyHeight).
		MaxHeight(cardBodyHeight)

	selectedBarStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.List.SelectedFg))
	unreadBarStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.List.UnreadFg)).
		Bold(true)

	contentWidth := max(m.ui.width-2*cardPadding, 0)

	padToWidth := func(text string, width int) string {
		if width <= 0 {
			return ""
		}
		textWidth := lipgloss.Width(text)
		if textWidth >= width {
			return text
		}
		return text + strings.Repeat(" ", width-textWidth)
	}

	// Show loading state
	if m.inbox.loading {
		body.WriteString("Loading inbox...")
		return m.renderListLayout("", body.String())
	}
	emptyMessage := ""
	if len(m.inbox.threads) == 0 {
		emptyMessage = "No messages"
	} else if m.displayCount() == 0 {
		if m.search.remoteLoading {
			emptyMessage = "Searching..."
		} else {
			emptyMessage = fmt.Sprintf("No results for \"%s\"", m.search.query)
		}
	}

	// Get visible thread range
	start, end := m.getVisibleThreadRange()

	// Thread list (card style) - only render visible range
	if emptyMessage != "" {
		body.WriteString(emptyMessage)
	} else {
		for i := start; i < end; i++ {
			threadIndex := m.threadIndexAt(i)
			if threadIndex < 0 || threadIndex >= len(m.inbox.threads) {
				continue
			}
			thread := m.inbox.threads[threadIndex]

			isSelected := i == m.inbox.cursor
			prefix := " "
			if isSelected {
				prefix = selectedBarStyle.Render("┃")
			} else if thread.Unread {
				prefix = unreadBarStyle.Render("│")
			}
			prefixWidth := lipgloss.Width(prefix)

			// Show loading state if metadata not loaded
			if !thread.Loaded {
				var cardContent strings.Builder
				lineWidth := max(contentWidth-prefixWidth, 0)

				// Line 1: Empty line (matches from + date line)
				cardContent.WriteString(prefix)
				cardContent.WriteString(strings.Repeat(" ", lineWidth))
				cardContent.WriteString("\n")

				// Line 2: Loading message (matches subject line)
				loadingText := "Loading..."
				cardContent.WriteString(prefix)
				cardContent.WriteString(padToWidth(loadingText, lineWidth))
				for line := 0; line < snippetLines; line++ {
					cardContent.WriteString("\n")
					cardContent.WriteString(prefix)
					cardContent.WriteString(strings.Repeat(" ", lineWidth))
				}

				body.WriteString(cardStyle.Render(cardContent.String()))
				body.WriteString("\n\n")
				continue
			}

			// Extract just the name from "Name <email>"
			from := thread.From
			// Simple extraction - just get text before '<' if present
			if idx := strings.Index(from, "<"); idx > 0 {
				from = strings.TrimSpace(from[:idx])
			}
			from = stripZeroWidth(from)
			if len(from) > 40 {
				from = from[:37] + "..."
			}

			// Format date and indicators
			date := formatRelativeTime(thread.Date)

			// Line 1: From + date/indicators
			indicatorsText := ""
			if thread.MessageCount > 1 {
				indicatorsText = fmt.Sprintf("(%d)", thread.MessageCount)
			}

			// Add account name if multiple accounts
			accountName := thread.AccountName
			if accountName == "" && thread.AccountIndex >= 0 && thread.AccountIndex < len(m.accountNames) {
				accountName = m.accountNames[thread.AccountIndex]
			}
			accountName = stripZeroWidth(accountName)
			accountText := ""
			accountStyle := lipgloss.Style{}
			if len(m.accountNames) > 1 && accountName != "" {
				badgeFg := m.theme.Status.TabFg
				badgeBg := m.theme.Status.TabBg
				if thread.AccountIndex >= 0 && thread.AccountIndex < len(m.accountBadges) {
					badge := m.accountBadges[thread.AccountIndex]
					if badge.Fg != "" {
						badgeFg = badge.Fg
					}
					if badge.Bg != "" {
						badgeBg = badge.Bg
					}
				}
				accountText = " " + accountName + " "
				accountStyle = lipgloss.NewStyle().
					Background(lipgloss.Color(badgeBg)).
					Foreground(lipgloss.Color(badgeFg)).
					Bold(true)
			}

			buildParts := func(indicators, account, date string) []rightPart {
				parts := []rightPart{}
				if indicators != "" {
					parts = append(parts, rightPart{text: indicators, style: dimStyle})
				}
				if account != "" {
					parts = append(parts, rightPart{text: account, style: accountStyle})
				}
				if date != "" {
					parts = append(parts, rightPart{text: date, style: dimStyle})
				}
				return parts
			}

			parts := buildParts(indicatorsText, accountText, date)

			maxRightWidth := max(contentWidth-prefixWidth, 0)
			if maxRightWidth > 0 && rightPartsWidth(parts) > maxRightWidth {
				accountText = ""
				parts = buildParts(indicatorsText, accountText, date)
			}
			if maxRightWidth > 0 && rightPartsWidth(parts) > maxRightWidth {
				indicatorsText = ""
				parts = buildParts(indicatorsText, accountText, date)
			}

			rightInfoRendered := ""
			if maxRightWidth == 0 {
				rightInfoRendered = ""
			} else {
				rightInfoRendered = renderRightParts(parts)
				if maxRightWidth > 0 && lipgloss.Width(rightInfoRendered) > maxRightWidth {
					rightInfoText := rightPartsText(parts)
					rightInfoText = truncateToWidth(rightInfoText, maxRightWidth)
					rightInfoRendered = dimStyle.Render(rightInfoText)
				}
			}

			availableFrom := max(contentWidth-prefixWidth-lipgloss.Width(rightInfoRendered), 0)
			fromMax := min(availableFrom, 40)
			if fromMax > 0 {
				from = truncateToWidth(from, fromMax)
			} else {
				from = ""
			}

			fromStyle := readStyle
			if thread.Unread {
				fromStyle = unreadStyle
			}

			line1Left := prefix + fromStyle.Render(from)
			if contentWidth > 0 {
				leftWidth := prefixWidth + lipgloss.Width(from)
				padding := contentWidth - leftWidth - lipgloss.Width(rightInfoRendered)
				if padding > 0 {
					line1Left += strings.Repeat(" ", padding)
				}
			}
			line1 := line1Left + rightInfoRendered

			// Style based on read/unread
			subjectStyle := readStyle
			if thread.Unread {
				subjectStyle = unreadStyle
			}

			// Line 2: Subject
			subject := thread.Subject
			subject = stripZeroWidth(subject)
			subjectWidth := max(contentWidth-prefixWidth, 0)
			if subjectWidth > 0 {
				subject = truncateToWidth(subject, subjectWidth)
			}
			line2 := prefix + subjectStyle.Render(subject)

			// Snippet lines
			snippet := thread.Snippet
			if subjectWidth > 0 {
				snippet = strings.TrimSpace(snippet)
			}
			snippet = stripZeroWidth(snippet)
			snippetText := wrapTextLines(snippet, subjectWidth, snippetLines)

			// Build card content
			var cardContent strings.Builder
			cardContent.WriteString(line1)
			cardContent.WriteString("\n")
			cardContent.WriteString(line2)
			lineSnippetStyle := snippetStyle
			if thread.Unread {
				lineSnippetStyle = unreadSnippetStyle
			}
			for lineIdx := 0; lineIdx < snippetLines; lineIdx++ {
				line := ""
				if lineIdx < len(snippetText) {
					line = snippetText[lineIdx]
				}
				line = stripLeadingZeroWidth(stripZeroWidth(line))
				line = padToWidth(line, subjectWidth)
				cardContent.WriteString("\n")
				cardContent.WriteString(prefix)
				cardContent.WriteString(lineSnippetStyle.Render(line))
			}

			body.WriteString(cardStyle.Render(cardContent.String()))
			body.WriteString("\n\n")
		}
	}

	return m.renderListLayout("", body.String())
}

func (m *Model) renderListLayout(header string, body string) string {
	footerLine := m.renderListStatusline()
	footerHeight := lipgloss.Height(footerLine)
	headerHeight := 0
	if header != "" {
		headerHeight = lipgloss.Height(header)
	}
	bodyHeight := max(0, m.ui.height-headerHeight-footerHeight)
	bodyStyle := lipgloss.NewStyle().Height(bodyHeight).MaxHeight(bodyHeight)
	body = bodyStyle.Render(body)

	if header == "" {
		return lipgloss.JoinVertical(lipgloss.Left, body, footerLine)
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footerLine)
}

func (m *Model) renderListStatusline() string {
	count := m.displayCount()
	pos := 0
	if count > 0 {
		pos = min(m.inbox.cursor+1, count)
	}
	total := len(m.inbox.threads)

	left := []statusSegment{}
	if m.search.active {
		left = append(left,
			statusModeSegment(m.theme, "SEARCH"),
			statusPaddedRaw(m.theme, m.search.input.View()),
		)
	} else {
		left = append(left, statusModeSegment(m.theme, "INBOX"))
		if m.search.query != "" {
			left = append(left, statusDimSegment(m.theme, "filter: "+m.search.query))
		}
	}

	if total > 0 && (m.search.query != "" || m.search.active) {
		left = append(left, statusTextSegment(m.theme, fmt.Sprintf("threads %d/%d", count, total)))
	} else {
		left = append(left, statusTextSegment(m.theme, fmt.Sprintf("threads %d", count)))
	}

	right := []statusSegment{}
	switch {
	case m.inbox.refreshing:
		right = append(right, statusDimSegment(m.theme, "refreshing"))
	case m.inbox.loadingMore:
		right = append(right, statusDimSegment(m.theme, "loading more"))
	case m.inbox.loadingThreads > 0:
		right = append(
			right,
			statusDimSegment(m.theme, fmt.Sprintf("loading %d", m.inbox.loadingThreads)),
		)
	case m.inbox.loading:
		right = append(right, statusDimSegment(m.theme, "loading"))
	case m.search.remoteLoading:
		right = append(right, statusDimSegment(m.theme, "searching"))
	}

	right = append(right, statusTextSegment(m.theme, fmt.Sprintf("%d/%d", pos, count)))

	if m.search.active {
		right = append(
			right,
			statusDimSegment(m.theme, "enter apply"),
			statusDimSegment(m.theme, "esc cancel"),
		)
	} else {
		right = append(right, statusDimSegment(m.theme, "? help"), statusDimSegment(m.theme, "q quit"))
	}

	return renderStatusline(m.theme, m.ui.width, left, right)
}
