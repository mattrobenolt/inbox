package tui

import (
	"strconv"
	"strings"

	"go.withmatt.com/inbox/internal/gmail"
)

func (m *Model) displayCount() int {
	if m.search.query == "" {
		return len(m.inbox.threads)
	}
	return len(m.filteredIndices())
}

func (m *Model) threadIndexAt(displayIndex int) int {
	if displayIndex < 0 {
		return -1
	}
	if m.search.query == "" {
		if displayIndex >= len(m.inbox.threads) {
			return -1
		}
		return displayIndex
	}
	filtered := m.filteredIndices()
	if displayIndex >= len(filtered) {
		return -1
	}
	idx := filtered[displayIndex]
	if idx < 0 || idx >= len(m.inbox.threads) {
		return -1
	}
	return idx
}

func (m *Model) selectedThreadIndex() int {
	return m.threadIndexAt(m.inbox.cursor)
}

func (m *Model) visibleThreadIndices() []int {
	start, end := m.getVisibleThreadRange()
	if end <= start {
		return nil
	}

	indices := make([]int, 0, end-start)
	for i := start; i < end; i++ {
		idx := m.threadIndexAt(i)
		if idx >= 0 {
			indices = append(indices, idx)
		}
	}
	return indices
}

func (m *Model) filteredIndices() []int {
	if m.search.query == "" {
		return nil
	}
	if len(m.inbox.filteredIdx) == 0 {
		return nil
	}
	out := m.inbox.filteredIdx[:0]
	for _, idx := range m.inbox.filteredIdx {
		if idx >= 0 && idx < len(m.inbox.threads) {
			out = append(out, idx)
		}
	}
	m.inbox.filteredIdx = out
	return out
}

func (m *Model) applyFilter(query string) {
	query = strings.TrimSpace(query)
	prevQuery := m.search.query
	if query != prevQuery && prevQuery != "" {
		m.pruneSearchOnly()
	}
	if query != prevQuery {
		m.search.remoteKeys = nil
	}
	m.search.query = query
	m.logf("Search applyFilter query=%q prev=%q", query, prevQuery)
	if query == "" {
		m.inbox.filteredIdx = nil
		m.search.remoteLoading = false
		m.search.remoteGeneration++
		m.search.remoteKeys = nil
		m.pruneSearchOnly()
		m.clampCursor()
		m.logf("Search cleared")
		return
	}

	if looksStructuredQuery(query) {
		m.inbox.filteredIdx = m.filterByRemoteKeys()
		m.logf("Search structured query matched=%d", len(m.inbox.filteredIdx))
		m.inbox.cursor = 0
		m.clampCursor()
		return
	}

	var selectedThreadID string
	selectedAccountIndex := -1
	if idx := m.threadIndexAt(m.inbox.cursor); idx >= 0 && idx < len(m.inbox.threads) {
		selectedThreadID = m.inbox.threads[idx].ThreadID
		selectedAccountIndex = m.inbox.threads[idx].AccountIndex
	}

	terms := strings.Fields(strings.ToLower(query))
	filtered := make([]int, 0, len(m.inbox.threads))
	for i, thread := range m.inbox.threads {
		if !thread.Loaded {
			continue
		}
		if threadMatches(thread, terms) {
			filtered = append(filtered, i)
		}
	}
	m.inbox.filteredIdx = filtered
	m.logf("Search local filter matched=%d", len(filtered))

	m.inbox.cursor = 0
	if selectedThreadID != "" {
		for displayIdx, idx := range filtered {
			if m.inbox.threads[idx].ThreadID == selectedThreadID &&
				m.inbox.threads[idx].AccountIndex == selectedAccountIndex {
				m.inbox.cursor = displayIdx
				break
			}
		}
	}
	m.clampCursor()
}

func (m *Model) filterByRemoteKeys() []int {
	if len(m.search.remoteKeys) == 0 {
		return nil
	}
	filtered := make([]int, 0, len(m.search.remoteKeys))
	for i, thread := range m.inbox.threads {
		if _, ok := m.search.remoteKeys[threadKey(thread.ThreadID, thread.AccountIndex)]; ok {
			filtered = append(filtered, i)
		}
	}
	return filtered
}

func (m *Model) clampCursor() {
	count := m.displayCount()
	if count == 0 {
		m.inbox.cursor = 0
		m.ensureCursorVisible()
		return
	}
	if m.inbox.cursor < 0 {
		m.inbox.cursor = 0
		m.ensureCursorVisible()
		return
	}
	if m.inbox.cursor >= count {
		m.inbox.cursor = count - 1
	}
	m.ensureCursorVisible()
}

func (m *Model) pruneSearchOnly() {
	if len(m.inbox.threads) == 0 {
		return
	}
	kept := make([]gmail.Thread, 0, len(m.inbox.threads))
	for _, thread := range m.inbox.threads {
		if thread.SearchOnly {
			continue
		}
		kept = append(kept, thread)
	}
	m.inbox.threads = kept
}

func threadKey(threadID string, accountIndex int) string {
	return threadID + ":" + strconv.Itoa(accountIndex)
}

func (m *Model) mergeSearchResults(results []gmail.Thread) []int {
	if len(results) == 0 {
		return nil
	}
	existing := make(map[string]struct{}, len(m.inbox.threads))
	for _, thread := range m.inbox.threads {
		existing[threadKey(thread.ThreadID, thread.AccountIndex)] = struct{}{}
	}

	var newIndices []int
	for _, thread := range results {
		key := threadKey(thread.ThreadID, thread.AccountIndex)
		if _, found := existing[key]; found {
			continue
		}
		thread.SearchOnly = true
		m.inbox.threads = append(m.inbox.threads, thread)
		newIndices = append(newIndices, len(m.inbox.threads)-1)
	}
	return newIndices
}

func threadMatches(thread gmail.Thread, terms []string) bool {
	if len(terms) == 0 {
		return true
	}
	haystack := strings.ToLower(strings.Join(
		[]string{thread.Subject, thread.From, thread.Snippet, thread.AccountName},
		" ",
	))
	for _, term := range terms {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

func looksStructuredQuery(query string) bool {
	for term := range strings.FieldsSeq(query) {
		if strings.Contains(term, ":") {
			return true
		}
	}
	return false
}
