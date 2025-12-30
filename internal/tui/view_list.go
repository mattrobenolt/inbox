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

func rightPartsText(parts []rightPart, lead string) string {
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
	return lead + b.String()
}

func renderRightParts(parts []rightPart, space string, lead string) string {
	var b strings.Builder
	first := true
	for _, part := range parts {
		if part.text == "" {
			continue
		}
		if !first {
			b.WriteString(space)
		}
		b.WriteString(part.style.Render(part.text))
		first = false
	}
	if b.Len() == 0 {
		return ""
	}
	return lead + b.String()
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
	selectedBg := strings.TrimSpace(m.theme.List.SelectedBg)
	bulkBg := lipgloss.Color(selectedBg)

	selectedBarStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.List.SelectedFg))
	bulkSelectedBarStyle := lipgloss.NewStyle().
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
			isBulkSelected := m.isThreadSelected(thread)
			useBulkBg := isBulkSelected && selectedBg != ""

			lineUnreadStyle := unreadStyle
			lineReadStyle := readStyle
			lineDimStyle := dimStyle
			lineSnippetStyle := snippetStyle
			lineUnreadSnippetStyle := unreadSnippetStyle
			lineSelectedBarStyle := selectedBarStyle
			lineBulkSelectedBarStyle := bulkSelectedBarStyle
			lineUnreadBarStyle := unreadBarStyle
			lineSpaceStyle := lipgloss.NewStyle()
			if useBulkBg {
				lineUnreadStyle = lineUnreadStyle.Background(bulkBg)
				lineReadStyle = lineReadStyle.Background(bulkBg)
				lineDimStyle = lineDimStyle.Background(bulkBg)
				lineSnippetStyle = lineSnippetStyle.Background(bulkBg)
				lineUnreadSnippetStyle = lineUnreadSnippetStyle.Background(bulkBg)
				lineSelectedBarStyle = lineSelectedBarStyle.Background(bulkBg)
				lineBulkSelectedBarStyle = lineBulkSelectedBarStyle.Background(bulkBg)
				lineUnreadBarStyle = lineUnreadBarStyle.Background(bulkBg)
				lineSpaceStyle = lineSpaceStyle.Background(bulkBg)
			}
			prefix := " "
			switch {
			case isSelected:
				prefix = lineSelectedBarStyle.Render("┃")
			case isBulkSelected:
				prefix = lineBulkSelectedBarStyle.Render("▌")
			case thread.Unread:
				prefix = lineUnreadBarStyle.Render("│")
			}
			prefixWidth := lipgloss.Width(prefix)
			suffix := " "
			if isBulkSelected {
				suffix = lineBulkSelectedBarStyle.Render("▐")
			} else if useBulkBg {
				suffix = lineSpaceStyle.Render(" ")
			}
			suffixWidth := lipgloss.Width(suffix)
			lineWidth := max(contentWidth-prefixWidth-suffixWidth, 0)
			space := " "
			if useBulkBg {
				space = lineSpaceStyle.Render(" ")
			}
			lead := space

			// Show loading state if metadata not loaded
			if !thread.Loaded {
				var cardContent strings.Builder

				// Line 1: Empty line (matches from + date line)
				cardContent.WriteString(prefix)
				blankLine := strings.Repeat(" ", lineWidth)
				if useBulkBg {
					blankLine = lineSpaceStyle.Render(blankLine)
				}
				cardContent.WriteString(blankLine)
				cardContent.WriteString(suffix)
				cardContent.WriteString("\n")

				// Line 2: Loading message (matches subject line)
				loadingText := "Loading..."
				cardContent.WriteString(prefix)
				loadingLine := padToWidth(loadingText, lineWidth)
				if useBulkBg {
					loadingLine = lineDimStyle.Render(loadingLine)
				}
				cardContent.WriteString(loadingLine)
				cardContent.WriteString(suffix)
				for line := 0; line < snippetLines; line++ {
					cardContent.WriteString("\n")
					cardContent.WriteString(prefix)
					blankLine = strings.Repeat(" ", lineWidth)
					if useBulkBg {
						blankLine = lineSpaceStyle.Render(blankLine)
					}
					cardContent.WriteString(blankLine)
					cardContent.WriteString(suffix)
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
				if useBulkBg {
					accountStyle = accountStyle.Background(bulkBg)
				}
			}

			buildParts := func(indicators, account, date string) []rightPart {
				parts := []rightPart{}
				if indicators != "" {
					parts = append(parts, rightPart{text: indicators, style: lineDimStyle})
				}
				if account != "" {
					parts = append(parts, rightPart{text: account, style: accountStyle})
				}
				if date != "" {
					parts = append(parts, rightPart{text: date, style: lineDimStyle})
				}
				return parts
			}

			parts := buildParts(indicatorsText, accountText, date)

			maxRightWidth := max(lineWidth, 0)
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
				rightInfoRendered = renderRightParts(parts, space, lead)
				if maxRightWidth > 0 && lipgloss.Width(rightInfoRendered) > maxRightWidth {
					rightInfoText := rightPartsText(parts, " ")
					rightInfoText = truncateToWidth(rightInfoText, maxRightWidth)
					rightInfoRendered = lineDimStyle.Render(rightInfoText)
				}
			}

			availableFrom := max(lineWidth-lipgloss.Width(rightInfoRendered), 0)
			fromMax := min(availableFrom, 40)
			if fromMax > 0 {
				from = truncateToWidth(from, fromMax)
			} else {
				from = ""
			}

			fromStyle := lineReadStyle
			if thread.Unread {
				fromStyle = lineUnreadStyle
			}

			line1Left := prefix + fromStyle.Render(from)
			if lineWidth > 0 {
				leftWidth := lipgloss.Width(from)
				padding := lineWidth - leftWidth - lipgloss.Width(rightInfoRendered)
				if padding > 0 {
					line1Left += strings.Repeat(space, padding)
				}
			}
			line1 := line1Left + rightInfoRendered + suffix

			// Style based on read/unread
			subjectStyle := lineReadStyle
			if thread.Unread {
				subjectStyle = lineUnreadStyle
			}

			// Line 2: Subject
			subject := thread.Subject
			subject = stripZeroWidth(subject)
			subjectWidth := max(lineWidth, 0)
			if subjectWidth > 0 {
				subject = truncateToWidth(subject, subjectWidth)
				subject = padToWidth(subject, subjectWidth)
			}
			line2 := prefix + subjectStyle.Render(subject) + suffix

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
			if thread.Unread {
				lineSnippetStyle = lineUnreadSnippetStyle
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
				cardContent.WriteString(suffix)
			}

			renderStyle := cardStyle
			if useBulkBg {
				renderStyle = renderStyle.Background(lipgloss.Color(selectedBg))
			}
			body.WriteString(renderStyle.Render(cardContent.String()))
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
	if selectedCount := m.selectedCount(); selectedCount > 0 {
		left = append(left, statusTextSegment(m.theme, fmt.Sprintf("selected %d", selectedCount)))
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
	case m.inbox.undo.inProgress:
		right = append(right, statusDimSegment(m.theme, "undoing"))
	case m.inbox.delete.inProgress:
		label := "deleting"
		switch m.inbox.delete.action {
		case deleteActionArchive:
			label = "archiving"
		case deleteActionTrash:
			label = "trashing"
		case deleteActionPermanent:
			label = "deleting"
		}
		right = append(right, statusDimSegment(m.theme, label))
	}

	right = append(right, statusTextSegment(m.theme, fmt.Sprintf("%d/%d", pos, count)))

	switch {
	case m.inbox.delete.pending:
		right = append(
			right,
			statusDimSegment(m.theme, "y confirm"),
			statusDimSegment(m.theme, "n cancel"),
		)
	case m.search.active:
		right = append(
			right,
			statusDimSegment(m.theme, "enter apply"),
			statusDimSegment(m.theme, "esc cancel"),
		)
	default:
		right = append(
			right,
			statusDimSegment(m.theme, "? help"),
			statusDimSegment(m.theme, "q quit"),
		)
	}

	return renderStatusline(m.theme, m.ui.width, left, right)
}
