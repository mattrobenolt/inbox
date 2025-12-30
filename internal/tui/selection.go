package tui

import "go.withmatt.com/inbox/internal/gmail"

func (m *Model) isThreadSelected(thread gmail.Thread) bool {
	if len(m.inbox.selected) == 0 {
		return false
	}
	_, ok := m.inbox.selected[threadKey(thread.ThreadID, thread.AccountIndex)]
	return ok
}

func (m *Model) selectedCount() int {
	return len(m.inbox.selected)
}

func (m *Model) toggleThreadSelection(idx int) {
	if idx < 0 || idx >= len(m.inbox.threads) {
		return
	}
	key := threadKey(m.inbox.threads[idx].ThreadID, m.inbox.threads[idx].AccountIndex)
	if _, ok := m.inbox.selected[key]; ok {
		delete(m.inbox.selected, key)
		return
	}
	if m.inbox.selected == nil {
		m.inbox.selected = make(map[string]struct{})
	}
	m.inbox.selected[key] = struct{}{}
}

func (m *Model) clearSelection() {
	if len(m.inbox.selected) == 0 {
		return
	}
	m.inbox.selected = make(map[string]struct{})
}

func (m *Model) pruneSelection() {
	if len(m.inbox.selected) == 0 {
		return
	}
	keep := make(map[string]struct{}, len(m.inbox.selected))
	for _, thread := range m.inbox.threads {
		key := threadKey(thread.ThreadID, thread.AccountIndex)
		if _, ok := m.inbox.selected[key]; ok {
			keep[key] = struct{}{}
		}
	}
	m.inbox.selected = keep
}

func (m *Model) selectedRefs() []threadRef {
	if len(m.inbox.selected) == 0 {
		return nil
	}
	refs := make([]threadRef, 0, len(m.inbox.selected))
	for _, thread := range m.inbox.threads {
		key := threadKey(thread.ThreadID, thread.AccountIndex)
		if _, ok := m.inbox.selected[key]; ok {
			refs = append(
				refs,
				threadRef{threadID: thread.ThreadID, accountIndex: thread.AccountIndex},
			)
		}
	}
	return refs
}

func (m *Model) selectionOrCurrent() []threadRef {
	if m.selectedCount() > 0 {
		return m.selectedRefs()
	}
	if idx := m.selectedThreadIndex(); idx >= 0 && idx < len(m.inbox.threads) {
		thread := m.inbox.threads[idx]
		return []threadRef{{threadID: thread.ThreadID, accountIndex: thread.AccountIndex}}
	}
	return nil
}

func (m *Model) threadForRef(ref threadRef) *gmail.Thread {
	for i := range m.inbox.threads {
		thread := &m.inbox.threads[i]
		if thread.ThreadID == ref.threadID && thread.AccountIndex == ref.accountIndex {
			return thread
		}
	}
	return nil
}

func (m *Model) removeThreadsByRefs(refs []threadRef) {
	if len(refs) == 0 {
		return
	}
	remove := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		remove[threadKey(ref.threadID, ref.accountIndex)] = struct{}{}
	}
	kept := make([]gmail.Thread, 0, len(m.inbox.threads))
	for _, thread := range m.inbox.threads {
		key := threadKey(thread.ThreadID, thread.AccountIndex)
		if _, ok := remove[key]; ok {
			continue
		}
		kept = append(kept, thread)
	}
	m.inbox.threads = kept
	m.pruneSelection()
}

func (m *Model) reapplyFilterPreserveCursor() {
	selectedKey := ""
	if idx := m.threadIndexAt(m.inbox.cursor); idx >= 0 && idx < len(m.inbox.threads) {
		thread := m.inbox.threads[idx]
		selectedKey = threadKey(thread.ThreadID, thread.AccountIndex)
	}

	m.applyFilter(m.search.query)

	if selectedKey != "" {
		for displayIdx := 0; displayIdx < m.displayCount(); displayIdx++ {
			idx := m.threadIndexAt(displayIdx)
			if idx < 0 || idx >= len(m.inbox.threads) {
				continue
			}
			thread := m.inbox.threads[idx]
			if threadKey(thread.ThreadID, thread.AccountIndex) == selectedKey {
				m.inbox.cursor = displayIdx
				break
			}
		}
	}
	m.clampCursor()
}
