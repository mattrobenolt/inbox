package tui

const (
	listHeaderHeight = 0
	listFooterHeight = 1
)

func (m *Model) listCardHeight() int {
	snippetLines := m.uiConfig.ListSnippetLines
	if snippetLines <= 0 {
		snippetLines = 1
	}
	return snippetLines + 3
}

func (m *Model) visibleCardCount() int {
	availableHeight := m.ui.height - listHeaderHeight - listFooterHeight
	if availableHeight <= 0 {
		return 0
	}
	return availableHeight / m.listCardHeight()
}

func (m *Model) ensureCursorVisible() {
	visibleCards := m.visibleCardCount()
	if visibleCards <= 0 {
		m.inbox.scrollOffset = 0
		return
	}

	count := m.displayCount()
	if count <= visibleCards {
		m.inbox.scrollOffset = 0
		return
	}

	maxOffset := max(count-visibleCards, 0)
	if m.inbox.scrollOffset < 0 {
		m.inbox.scrollOffset = 0
	} else if m.inbox.scrollOffset > maxOffset {
		m.inbox.scrollOffset = maxOffset
	}

	if m.inbox.cursor < m.inbox.scrollOffset {
		m.inbox.scrollOffset = m.inbox.cursor
	} else if m.inbox.cursor >= m.inbox.scrollOffset+visibleCards {
		m.inbox.scrollOffset = m.inbox.cursor - visibleCards + 1
	}

	if m.inbox.scrollOffset < 0 {
		m.inbox.scrollOffset = 0
	} else if m.inbox.scrollOffset > maxOffset {
		m.inbox.scrollOffset = maxOffset
	}
}
