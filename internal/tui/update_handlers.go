package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"go.withmatt.com/inbox/internal/gmail"
)

func (m Model) updateSpinner(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.ui.spinner, cmd = m.ui.spinner.Update(msg)
	// Keep spinner ticking if we're downloading or loading an attachment preview
	if m.attachments.modal.downloading || m.attachments.modal.loadingPreview {
		return m, cmd
	}
	return m, cmd
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m = m.clearAlerts()
	// Close modals on any keypress (except navigation in attachments modal)
	if m.ui.showHelp {
		m.ui.showHelp = false
		return m, nil
	}
	if m.ui.showError {
		m.ui.showError = false
		m.ui.err = nil
		return m, nil
	}
	// Handle attachments modal separately since it needs navigation
	if m.attachments.modal.show {
		return m.handleAttachmentsModalKey(msg)
	}
	if m.search.active {
		return m.handleSearchKey(msg)
	}

	switch m.currentView {
	case viewList:
		return m.handleListKey(msg)
	case viewDetail:
		return m.handleDetailKey(msg)
	case viewAttachment:
		return m.handleAttachmentKey(msg)
	case viewImage:
		return m.handleImageKey(msg)
	default:
		return m, nil
	}
}

func (m Model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if !m.ui.focused {
		return m, nil
	}

	switch m.currentView {
	case viewList:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if m.inbox.cursor > 0 {
				m.inbox.cursor--
				m.ensureCursorVisible()
				// Load visible threads when scrolling
				return m, m.loadVisibleThreadsCmd()
			}
		case tea.MouseButtonWheelDown:
			if m.inbox.cursor < m.displayCount()-1 {
				m.inbox.cursor++
				m.ensureCursorVisible()
				// Load visible threads when scrolling
				cmd := m.loadVisibleThreadsCmd()

				// If near bottom (within 10 threads), trigger loading more
				if count := m.displayCount(); count > 0 &&
					m.inbox.cursor >= count-10 && m.inbox.nextPageToken != "" && !m.inbox.loadingMore {
					m.inbox.loadingMore = true
					return m, tea.Batch(cmd, m.loadMoreThreadsCmd())
				}
				return m, cmd
			}
		case tea.MouseButtonLeft:
			if msg.Action == tea.MouseActionPress {
				// Calculate which thread was clicked based on Y position
				// Each card is listCardHeight lines (including the blank separator).
				clickedCardIndex := (msg.Y - listHeaderHeight) / m.listCardHeight()

				// Map clicked card to actual thread index using visible range
				start, end := m.getVisibleThreadRange()
				actualIndex := start + clickedCardIndex

				if actualIndex >= start && actualIndex < end {
					m.inbox.cursor = actualIndex
					m.ensureCursorVisible()
					if idx := m.selectedThreadIndex(); idx >= 0 && idx < len(m.inbox.threads) {
						thread := m.inbox.threads[idx]
						return m, m.openThread(thread)
					}
					return m, nil
				}
			}
		case tea.MouseButtonNone,
			tea.MouseButtonMiddle,
			tea.MouseButtonRight,
			tea.MouseButtonWheelLeft,
			tea.MouseButtonWheelRight,
			tea.MouseButtonBackward,
			tea.MouseButtonForward,
			tea.MouseButton10,
			tea.MouseButton11:
			return m, nil
		}
	case viewDetail:
		// Pass mouse events to viewport for scrolling
		var cmd tea.Cmd
		m.detail.viewport, cmd = m.detail.viewport.Update(msg)
		return m, cmd
	case viewAttachment, viewImage:
		return m, nil
	}

	return m, nil
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	km := m.keyMap()
	if m.inbox.delete.pending {
		switch msg.String() {
		case "y", "Y", "enter":
			refs := append([]threadRef(nil), m.inbox.delete.targets...)
			action := m.inbox.delete.action
			m.inbox.delete.pending = false
			m.inbox.delete.targets = nil
			m.inbox.delete.action = deleteActionTrash
			if len(refs) == 0 {
				return m, nil
			}
			m.inbox.delete.inProgress = true
			return m, m.threadActionCmd(action, refs)
		case "n", "N", "esc":
			m.inbox.delete.pending = false
			m.inbox.delete.targets = nil
			m.inbox.delete.action = deleteActionTrash
			return m, nil
		default:
			return m, nil
		}
	}
	switch {
	case key.Matches(msg, km.list.Quit):
		return m, tea.Quit
	case key.Matches(msg, km.list.Help):
		m.ui.showHelp = true
		return m, nil
	case key.Matches(msg, km.list.Refresh):
		// Refresh inbox (non-disruptive)
		tea.Printf("USER: Pressed 'r' to refresh, current threads=%d", len(m.inbox.threads))
		m.inbox.refreshing = true
		return m, m.loadInboxCmd(inboxLoadManual)
	case key.Matches(msg, km.list.ToggleRead):
		// Toggle read/unread status
		if idx := m.selectedThreadIndex(); idx >= 0 && idx < len(m.inbox.threads) {
			thread := &m.inbox.threads[idx]
			// Calculate new state before optimistic update
			newUnreadState := !thread.Unread
			// Optimistically update UI immediately
			thread.Unread = newUnreadState
			// Send API request in background with the target state
			return m, m.markThreadUnreadCmd(thread.ThreadID, newUnreadState, thread.AccountIndex)
		}
	case key.Matches(msg, km.list.ToggleSelect):
		if idx := m.selectedThreadIndex(); idx >= 0 && idx < len(m.inbox.threads) {
			m.toggleThreadSelection(idx)
		}
		return m, nil
	case key.Matches(msg, km.list.ClearSelection):
		m.clearSelection()
		return m, nil
	case key.Matches(msg, km.list.Archive):
		refs := m.selectionOrCurrent()
		if len(refs) == 0 {
			return m, nil
		}
		m.inbox.delete.pending = true
		m.inbox.delete.targets = refs
		m.inbox.delete.action = deleteActionArchive
		return m, nil
	case key.Matches(msg, km.list.Delete):
		refs := m.selectionOrCurrent()
		if len(refs) == 0 {
			return m, nil
		}
		m.inbox.delete.pending = true
		m.inbox.delete.targets = refs
		m.inbox.delete.action = deleteActionTrash
		return m, nil
	case key.Matches(msg, km.list.DeleteForever):
		refs := m.selectionOrCurrent()
		if len(refs) == 0 {
			return m, nil
		}
		m.inbox.delete.pending = true
		m.inbox.delete.targets = refs
		m.inbox.delete.action = deleteActionPermanent
		return m, nil
	case key.Matches(msg, km.list.Undo):
		if !m.undoAvailable() || m.inbox.undo.inProgress {
			return m, nil
		}
		refs := m.undoRefs()
		if len(refs) == 0 {
			return m, nil
		}
		action := m.inbox.undo.action
		m.inbox.undo.inProgress = true
		m = m.clearAlerts()
		return m, m.undoThreadsCmd(action, refs)
	case key.Matches(msg, km.list.PageUp):
		// Jump up by visible page size
		start, end := m.getVisibleThreadRange()
		pageSize := end - start
		if pageSize > 0 {
			m.inbox.cursor = max(0, m.inbox.cursor-pageSize)
			m.ensureCursorVisible()
			return m, m.loadVisibleThreadsCmd()
		}
	case key.Matches(msg, km.list.PageDown):
		// Jump down by visible page size
		start, end := m.getVisibleThreadRange()
		pageSize := end - start
		count := m.displayCount()
		if pageSize > 0 {
			if count > 0 {
				m.inbox.cursor = min(count-1, m.inbox.cursor+pageSize)
			}
			m.ensureCursorVisible()
			cmd := m.loadVisibleThreadsCmd()

			// If near bottom, trigger loading more
			if count > 0 && m.inbox.cursor >= count-10 && m.inbox.nextPageToken != "" &&
				!m.inbox.loadingMore {
				m.inbox.loadingMore = true
				return m, tea.Batch(cmd, m.loadMoreThreadsCmd())
			}
			return m, cmd
		}
	case key.Matches(msg, km.list.Up):
		if m.inbox.cursor > 0 {
			m.inbox.cursor--
			m.ensureCursorVisible()
			// Load visible threads when scrolling
			return m, m.loadVisibleThreadsCmd()
		}
	case key.Matches(msg, km.list.Down):
		if m.inbox.cursor < m.displayCount()-1 {
			m.inbox.cursor++
			m.ensureCursorVisible()
			// Load visible threads when scrolling
			cmd := m.loadVisibleThreadsCmd()

			// If near bottom (within 10 threads), trigger loading more
			if count := m.displayCount(); count > 0 &&
				m.inbox.cursor >= count-10 && m.inbox.nextPageToken != "" && !m.inbox.loadingMore {
				m.inbox.loadingMore = true
				return m, tea.Batch(cmd, m.loadMoreThreadsCmd())
			}
			return m, cmd
		}
	case key.Matches(msg, km.list.Open):
		if idx := m.selectedThreadIndex(); idx >= 0 && idx < len(m.inbox.threads) {
			thread := m.inbox.threads[idx]
			return m, m.openThread(thread)
		}
	case key.Matches(msg, km.list.Search):
		m.search.previousQuery = m.search.query
		m.search.active = true
		m.search.input.SetValue(m.search.query)
		m.search.input.CursorEnd()
		m.search.input.Focus()
		m.search.input.Width = max(10, m.ui.width-4)
		m.logf("Search open query=%q", m.search.query)
		return m, textinput.Blink
	}

	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	km := m.keyMap()
	switch {
	case key.Matches(msg, km.search.Quit):
		return m, tea.Quit
	case key.Matches(msg, km.search.Cancel):
		m.search.active = false
		m.search.input.Blur()
		m.search.input.SetValue(m.search.previousQuery)
		m.applyFilter(m.search.previousQuery)
		m.logf("Search cancel restore=%q", m.search.previousQuery)
		return m, nil
	case key.Matches(msg, km.search.Submit):
		query := strings.TrimSpace(m.search.input.Value())
		m.search.active = false
		m.search.input.Blur()
		m.applyFilter(query)
		m.logf("Search submit query=%q", query)
		gen := m.search.remoteGeneration + 1
		m.search.remoteGeneration = gen
		if query == "" {
			m.search.remoteLoading = false
			m.pruneSearchOnly()
			return m, nil
		}
		return m, m.searchDebounceCmd(query, gen)
	}

	var cmd tea.Cmd
	m.search.input, cmd = m.search.input.Update(msg)
	m.applyFilter(m.search.input.Value())
	gen := m.search.remoteGeneration + 1
	m.search.remoteGeneration = gen
	query := strings.TrimSpace(m.search.input.Value())
	if query == "" {
		m.search.remoteLoading = false
		m.pruneSearchOnly()
		m.logf("Search typing empty")
		return m, cmd
	}
	m.logf("Search typing query=%q gen=%d", query, gen)
	return m, tea.Batch(cmd, m.searchDebounceCmd(query, gen))
}

func (m Model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	km := m.keyMap()
	switch {
	case key.Matches(msg, km.detail.Quit):
		return m, tea.Quit
	case msg.String() == "X":
		m.debugDumpCurrentMessage()
		return m, nil
	case key.Matches(msg, km.detail.Help):
		m.ui.showHelp = true
		return m, nil
	case key.Matches(msg, km.detail.Attachments):
		// Show attachments modal if current message has attachments
		if m.detail.selectedMessageIdx >= 0 &&
			m.detail.selectedMessageIdx < len(m.detail.messages) {
			msg := m.detail.messages[m.detail.selectedMessageIdx]
			if len(msg.Attachments) > 0 {
				m.showAttachmentsModal(msg)
				return m, m.setWindowTitleCmd()
			}
		}
		return m, nil
	case key.Matches(msg, km.detail.Back):
		// Go back to list view
		return m, m.exitDetailView()
	case key.Matches(msg, km.detail.ToggleView):
		// Toggle view mode for the selected message.
		if m.detail.selectedMessageIdx >= 0 &&
			m.detail.selectedMessageIdx < len(m.detail.messages) {
			selected := m.detail.messages[m.detail.selectedMessageIdx]
			m.detail.messageViewMode = nextMessageViewMode(m.detail.messageViewMode, selected)
		}
		var rawCmd tea.Cmd
		if m.detail.messageViewMode == viewModeRaw {
			rawCmd = m.loadRawForExpandedMessages()
		}
		// Re-render the message and reset viewport position
		body := m.renderThreadBody()
		m.detail.viewport.SetContent(body)
		m.detail.viewport.GotoTop()
		m.detail.viewport.YOffset = 0
		// Force full redraw
		if rawCmd != nil {
			return m, tea.Batch(tea.ClearScreen, rawCmd)
		}
		return m, tea.ClearScreen
	case key.Matches(msg, km.detail.Down):
		if m.detail.selectedMessageIdx < len(m.detail.messages)-1 {
			m.detail.selectedMessageIdx++
			// Re-render
			body := m.renderThreadBody()
			m.detail.viewport.SetContent(body)
		}
	case key.Matches(msg, km.detail.Up):
		if m.detail.selectedMessageIdx > 0 {
			m.detail.selectedMessageIdx--
			// Re-render
			body := m.renderThreadBody()
			m.detail.viewport.SetContent(body)
		}
	case key.Matches(msg, km.detail.ToggleExpand):
		// Toggle expand/collapse for selected message
		if m.detail.selectedMessageIdx >= 0 &&
			m.detail.selectedMessageIdx < len(m.detail.messages) {
			msgID := m.detail.messages[m.detail.selectedMessageIdx].ID
			m.detail.expandedMessages[msgID] = !m.detail.expandedMessages[msgID]
			var rawCmd tea.Cmd
			if m.detail.messageViewMode == viewModeRaw {
				rawCmd = m.loadRawForExpandedMessages()
			}
			// Re-render
			body := m.renderThreadBody()
			m.detail.viewport.SetContent(body)
			if rawCmd != nil {
				return m, rawCmd
			}
		}
	}

	var cmd tea.Cmd
	m.detail.viewport, cmd = m.detail.viewport.Update(msg)
	return m, cmd
}

func (m Model) handleAttachmentKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	km := m.keyMap()
	switch {
	case key.Matches(msg, km.attachment.Quit):
		return m, tea.Quit
	case key.Matches(msg, km.attachment.Back):
		return m, m.exitAttachmentView()
	}

	var cmd tea.Cmd
	m.detail.viewport, cmd = m.detail.viewport.Update(msg)
	return m, cmd
}

func (m Model) handleImageKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	km := m.keyMap()
	switch {
	case key.Matches(msg, km.image.Quit):
		return m, tea.Quit
	case key.Matches(msg, km.image.Back):
		return m, m.exitImageView()
	}

	return m, nil
}

func (m Model) handleInboxLoaded(msg inboxLoadedMsg) (tea.Model, tea.Cmd) {
	tea.Printf(
		"INBOX: Received %d threads from API, existing=%d",
		len(msg.threads),
		len(m.inbox.threads),
	)
	m.inbox.loading = false
	m.inbox.refreshing = false
	m.inbox.loadingMore = false
	if msg.err != nil {
		m.ui.err = msg.err
		m.ui.showError = true
		return m, nil
	}

	// Store pagination token
	m.inbox.nextPageToken = msg.nextPageToken

	// If append mode (pagination), just add to end
	if msg.append {
		tea.Printf("INBOX: Append mode, adding to end")
		m.inbox.threads = append(m.inbox.threads, msg.threads...)
		// Load metadata for newly added threads if visible
		return m, m.loadVisibleThreadsCmd()
	}

	// If this was a refresh, merge new threads with existing ones
	added := 0
	if len(m.inbox.threads) > 0 {
		tea.Printf("INBOX: Reconciling - merging with %d existing threads", len(m.inbox.threads))
		// Build a map of existing threads by ID for quick lookup
		existingThreads := make(map[string]gmail.Thread)
		for _, thread := range m.inbox.threads {
			existingThreads[thread.ThreadID] = thread
		}

		// Merge: preserve existing threads, add new threads
		kept := 0
		var mergedThreads []gmail.Thread
		for _, newThread := range msg.threads {
			if existingThread, found := existingThreads[newThread.ThreadID]; found {
				// Keep the existing thread (it has loaded metadata)
				// But update account info from the fresh thread (in case account order changed)
				existingThread.AccountIndex = newThread.AccountIndex
				existingThread.AccountName = newThread.AccountName
				mergedThreads = append(mergedThreads, existingThread)
				delete(existingThreads, newThread.ThreadID)
				kept++
			} else {
				// New thread - add it (will need metadata loaded)
				mergedThreads = append(mergedThreads, newThread)
				added++
			}
		}

		// Add remaining existing threads that weren't in the refresh
		remaining := 0
		for _, thread := range m.inbox.threads {
			if _, stillExists := existingThreads[thread.ThreadID]; stillExists {
				mergedThreads = append(mergedThreads, thread)
				remaining++
			}
		}

		tea.Printf(
			"INBOX: Reconciliation complete - kept=%d, added=%d, remaining=%d, total=%d",
			kept,
			added,
			remaining,
			len(mergedThreads),
		)
		m.inbox.threads = mergedThreads
	} else {
		// Initial load - just use the new threads
		tea.Printf("INBOX: Initial load, no existing threads to merge")
		m.inbox.threads = msg.threads
	}

	// Ensure cursor is still valid
	if m.inbox.cursor >= len(m.inbox.threads) {
		m.inbox.cursor = len(m.inbox.threads) - 1
	}
	if m.inbox.cursor < 0 && len(m.inbox.threads) > 0 {
		m.inbox.cursor = 0
	}

	// Count threads that still need metadata so we can decide whether to sort now.
	needsLoading := 0
	for i := range m.inbox.threads {
		if !m.inbox.threads[i].Loaded {
			needsLoading++
		}
	}
	if needsLoading == 0 {
		sortThreadsByDate(m.inbox.threads)
	}
	if m.search.query != "" {
		m.applyFilter(m.search.query)
	} else {
		m.inbox.filteredIdx = nil
		m.clampCursor()
	}
	m.pruneSelection()

	notify := msg.source == inboxLoadAuto && added > 0

	// Only load metadata if threads don't already have it (not from cache)
	tea.Printf(
		"INBOX: After reconcile - total=%d, needsLoading=%d",
		len(m.inbox.threads),
		needsLoading,
	)

	// Load metadata for threads that need it
	if needsLoading > 0 {
		tea.Printf("INBOX: Starting metadata load for %d threads", needsLoading)
		cmd := m.loadAllThreadsMetadataCmd(false)
		if notify {
			return m, tea.Batch(cmd, bellCmd())
		}
		return m, cmd
	}

	tea.Printf("INBOX: No threads need loading")
	if notify {
		return m, bellCmd()
	}
	return m, nil
}

func (m Model) handleThreadMetadataLoaded(msg threadMetadataLoadedMsg) Model {
	if msg.err != nil {
		// Silently skip - thread will show as "Loading..."
		tea.Printf("METADATA: Failed to load thread at index %d: %v", msg.index, msg.err)
		return m
	}
	// Update the thread at the specified index with loaded metadata
	if msg.thread != nil {
		targetIndex := findThreadIndex(m.inbox.threads, msg.accountIndex, msg.threadID)
		if targetIndex == -1 {
			m.logf(
				"Metadata update skipped account=%d thread=%s (not found)",
				msg.accountIndex,
				msg.threadID,
			)
			return m
		}
		// Preserve account info when updating metadata
		msg.thread.AccountIndex = m.inbox.threads[targetIndex].AccountIndex
		msg.thread.AccountName = m.inbox.threads[targetIndex].AccountName
		m.inbox.threads[targetIndex] = *msg.thread
		m.inbox.loadedThreads++

		threadIDShort := msg.thread.ThreadID
		if len(threadIDShort) > 8 {
			threadIDShort = threadIDShort[:8]
		}
		tea.Printf(
			"METADATA: Loaded thread %d/%d (ID=%s, Subject=%s)",
			m.inbox.loadedThreads,
			len(m.inbox.threads),
			threadIDShort,
			msg.thread.Subject,
		)

		// Re-sort threads after each update for streaming effect
		sortThreadsByDate(m.inbox.threads)
		if m.search.query != "" {
			m.applyFilter(m.search.query)
		}
	}
	return m
}

func (m Model) handleBatchLoadStart(msg batchLoadStartMsg) Model {
	// Track how many threads we're loading
	m.inbox.loadingThreads += msg.count
	return m
}

func (m Model) handleBatchThreadMetadataLoaded(
	msg batchThreadMetadataLoadedMsg,
) Model {
	// Update all threads from batch load
	for _, result := range msg.results {
		if result.err != nil {
			// Silently skip - thread will show as "Loading..."
			m.inbox.loadingThreads--
			continue
		}
		if result.thread != nil {
			targetIndex := findThreadIndex(m.inbox.threads, result.accountIndex, result.threadID)
			if targetIndex == -1 {
				m.logf(
					"Batch metadata update skipped account=%d thread=%s (not found)",
					result.accountIndex,
					result.threadID,
				)
				m.inbox.loadingThreads--
				continue
			}
			// Preserve account info when updating metadata
			result.thread.AccountIndex = m.inbox.threads[targetIndex].AccountIndex
			result.thread.AccountName = m.inbox.threads[targetIndex].AccountName
			m.inbox.threads[targetIndex] = *result.thread
			m.inbox.loadedThreads++
			m.inbox.loadingThreads--
		}
	}
	// Re-sort threads by date after loading metadata
	sortThreadsByDate(m.inbox.threads)
	if m.search.query != "" {
		m.applyFilter(m.search.query)
	}
	return m
}

func (m Model) handleThreadLoaded(msg threadLoadedMsg) (tea.Model, tea.Cmd) {
	m.detail.loading = false
	if msg.err != nil {
		if m.detail.currentThread != nil {
			m.logf(
				"GetThread failed account=%d thread=%s err=%v",
				m.detail.currentThread.AccountIndex,
				m.detail.currentThread.ThreadID,
				msg.err,
			)
		}
		m.ui.err = msg.err
		m.ui.showError = true
		return m, nil
	}
	// Reverse messages so newest is first
	m.detail.messages = reverseMessages(msg.messages)

	// Expand the first (most recent) message by default
	m.detail.expandedMessages = make(map[string]bool)
	if len(m.detail.messages) > 0 {
		m.detail.expandedMessages[m.detail.messages[0].ID] = true
	}
	m.detail.selectedMessageIdx = 0

	var rawCmd tea.Cmd
	if m.detail.messageViewMode == viewModeRaw {
		rawCmd = m.loadRawForExpandedMessages()
	}

	// Update viewport size first
	m.detail.viewport.Width = m.ui.width
	m.detail.viewport.Height = detailViewportHeight(m.ui.height)
	// Set viewport content
	body := m.renderThreadBody()
	m.detail.viewport.SetContent(body)
	// Reset scroll position to top
	m.detail.viewport.GotoTop()
	return m, rawCmd
}

func (m Model) handleThreadMarked(msg threadMarkedMsg) Model {
	if msg.err != nil {
		// Revert optimistic update on error
		for i := range m.inbox.threads {
			if m.inbox.threads[i].ThreadID == msg.threadID {
				m.inbox.threads[i].Unread = !msg.unread // Revert to opposite of what we tried to set
				break
			}
		}
		if m.detail.currentThread != nil && m.detail.currentThread.ThreadID == msg.threadID {
			m.detail.currentThread.Unread = !msg.unread
		}
		m.ui.err = msg.err
		m.ui.showError = true
		return m
	}
	// Success - the optimistic update was correct, no need to update again
	return m
}

func (m Model) handleThreadsAction(msg threadsActionMsg) (tea.Model, tea.Cmd) {
	m.inbox.delete.inProgress = false
	m.inbox.delete.action = deleteActionTrash
	m.inbox.undo.inProgress = false
	if len(msg.refs) == 0 {
		return m, nil
	}

	failed := make(map[string]struct{}, len(msg.failed))
	for _, ref := range msg.failed {
		failed[threadKey(ref.threadID, ref.accountIndex)] = struct{}{}
	}

	succeeded := make([]threadRef, 0, len(msg.refs)-len(msg.failed))
	for _, ref := range msg.refs {
		if _, ok := failed[threadKey(ref.threadID, ref.accountIndex)]; ok {
			continue
		}
		succeeded = append(succeeded, ref)
	}

	var undoThreads []gmail.Thread
	if len(succeeded) > 0 {
		undoThreads = m.threadsForRefs(succeeded)
		m.removeThreadsByRefs(succeeded)
		if m.search.query != "" {
			m.reapplyFilterPreserveCursor()
		} else {
			m.clampCursor()
		}
	}

	if msg.err != nil {
		m.ui.err = msg.err
		m.ui.showError = true
	}

	switch msg.action {
	case deleteActionArchive, deleteActionTrash:
		if len(undoThreads) > 0 {
			m.inbox.undo = undoState{action: msg.action, threads: undoThreads}
		} else {
			m.inbox.undo = undoState{}
		}
	case deleteActionPermanent:
		m.inbox.undo = undoState{}
	}

	var toastCmd tea.Cmd
	if msg.err == nil && len(undoThreads) > 0 {
		toastCmd = m.undoToastCmd(msg.action, len(undoThreads))
	}

	return m, toastCmd
}

func (m Model) handleThreadsUndo(msg threadsUndoMsg) Model {
	m.inbox.undo.inProgress = false
	if len(msg.refs) == 0 {
		return m
	}

	failed := make(map[string]struct{}, len(msg.failed))
	for _, ref := range msg.failed {
		failed[threadKey(ref.threadID, ref.accountIndex)] = struct{}{}
	}

	reinsert := make([]gmail.Thread, 0, len(msg.refs)-len(msg.failed))
	remainingUndo := make([]gmail.Thread, 0, len(msg.failed))
	for _, thread := range m.inbox.undo.threads {
		key := threadKey(thread.ThreadID, thread.AccountIndex)
		if _, ok := failed[key]; ok {
			remainingUndo = append(remainingUndo, thread)
			continue
		}
		reinsert = append(reinsert, thread)
	}

	if len(reinsert) > 0 {
		m.inbox.threads = append(m.inbox.threads, reinsert...)
		if m.search.query != "" {
			m.reapplyFilterPreserveCursor()
		} else {
			sortThreadsByDate(m.inbox.threads)
			m.clampCursor()
		}
	}

	if len(remainingUndo) > 0 {
		m.inbox.undo.threads = remainingUndo
	} else {
		m.inbox.undo = undoState{}
	}

	if msg.err != nil {
		m.ui.err = msg.err
		m.ui.showError = true
	}

	return m
}

func (m Model) handleAttachmentDownloaded(msg attachmentDownloadedMsg) Model {
	// Stop downloading state
	m.attachments.modal.downloading = false
	// Close attachments modal
	m.attachments.modal.show = false
	m.attachments.modal.attachments = nil
	m.attachments.modal.selectedIdx = 0

	if msg.err != nil {
		// Show error modal
		m.ui.err = fmt.Errorf("failed to download %s: %w", msg.filename, msg.err)
		m.ui.showError = true
	}
	// On success, just close the modal silently
	return m
}

func (m Model) handleAttachmentLoaded(msg attachmentLoadedMsg) (tea.Model, tea.Cmd) {
	// Stop loading state
	m.attachments.modal.loadingPreview = false

	if msg.err != nil {
		// Show error modal
		m.ui.err = fmt.Errorf("failed to load %s: %w", msg.filename, msg.err)
		m.ui.showError = true
		return m, nil
	}

	switch {
	case isImageMimeType(msg.mimeType):
		m.enterImageView(msg)
	case isTextAttachment(msg.mimeType, msg.filename):
		if err := m.enterAttachmentView(msg); err != nil {
			m.ui.err = err
			m.ui.showError = true
			return m, nil
		}
	default:
		m.ui.err = fmt.Errorf("unsupported attachment type: %s", msg.mimeType)
		m.ui.showError = true
		return m, nil
	}

	// Close attachments modal
	m.attachments.modal.show = false

	return m, m.setWindowTitleCmd()
}

func (m Model) handleMessageRawLoaded(msg messageRawLoadedMsg) Model {
	delete(m.detail.rawLoading, msg.messageID)
	if msg.err != nil {
		if m.detail.currentThread != nil && m.detail.currentThread.ThreadID == msg.threadID {
			m.ui.err = msg.err
			m.ui.showError = true
		}
		return m
	}
	if m.detail.currentThread == nil || m.detail.currentThread.ThreadID != msg.threadID {
		return m
	}
	for i := range m.detail.messages {
		if m.detail.messages[i].ID == msg.messageID {
			m.detail.messages[i].Raw = msg.raw
			break
		}
	}
	if m.currentView == viewDetail {
		body := m.renderThreadBody()
		m.detail.viewport.SetContent(body)
	}
	return m
}

func (m Model) handleSearchDebounce(msg searchDebounceMsg) (tea.Model, tea.Cmd) {
	if msg.generation != m.search.remoteGeneration {
		m.logf(
			"Search debounce skipped stale gen=%d current=%d",
			msg.generation,
			m.search.remoteGeneration,
		)
		return m, nil
	}
	if strings.TrimSpace(msg.query) != m.search.query {
		m.logf("Search debounce skipped query=%q current=%q", msg.query, m.search.query)
		return m, nil
	}
	if msg.query == "" {
		m.search.remoteLoading = false
		m.logf("Search debounce empty")
		return m, nil
	}
	m.search.remoteLoading = true
	m.logf("Search debounce fire query=%q gen=%d", msg.query, msg.generation)
	return m, m.searchRemoteCmd(msg.query, msg.generation)
}

func (m Model) handleSearchRemoteLoaded(msg searchRemoteLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.generation != m.search.remoteGeneration {
		m.logf(
			"Search remote drop stale gen=%d current=%d",
			msg.generation,
			m.search.remoteGeneration,
		)
		return m, nil
	}
	if strings.TrimSpace(msg.query) != m.search.query {
		m.logf("Search remote drop query=%q current=%q", msg.query, m.search.query)
		return m, nil
	}
	m.search.remoteLoading = false
	m.logf("Search remote loaded query=%q threads=%d err=%v", msg.query, len(msg.threads), msg.err)

	if msg.err != nil {
		m.ui.err = msg.err
		m.ui.showError = true
		return m, nil
	}

	m.search.remoteKeys = make(map[string]struct{}, len(msg.threads))
	for _, thread := range msg.threads {
		m.search.remoteKeys[threadKey(thread.ThreadID, thread.AccountIndex)] = struct{}{}
	}

	m.pruneSearchOnly()
	newIndices := m.mergeSearchResults(msg.threads)
	if m.search.query != "" {
		m.applyFilter(m.search.query)
	}
	m.logf("Search remote merged new=%d total=%d", len(newIndices), len(m.inbox.threads))
	return m, m.loadThreadsMetadataCmd(newIndices)
}

func (m Model) handleAutoRefresh(msg autoRefreshMsg) (tea.Model, tea.Cmd) {
	if m.uiConfig.RefreshIntervalSeconds <= 0 {
		return m, nil
	}
	if m.inbox.loading || m.inbox.refreshing || m.inbox.loadingMore {
		return m, m.autoRefreshCmd()
	}
	m.inbox.refreshing = true
	return m, tea.Batch(m.loadInboxCmd(inboxLoadAuto), m.autoRefreshCmd())
}

func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	oldWidth := m.ui.width
	m.ui.width = msg.Width
	m.ui.height = msg.Height
	m.search.input.Width = max(10, msg.Width-4)
	if msg.Width != oldWidth && msg.Width > 0 {
		m.ui.alert = newAlertModel(m.theme, msg.Width)
	}

	// Update viewport size for detail view
	if m.currentView == viewDetail {
		m.detail.viewport.Width = msg.Width
		m.detail.viewport.Height = detailViewportHeight(msg.Height)
	}
	if m.currentView == viewAttachment {
		m.detail.viewport.Width = msg.Width
		m.detail.viewport.Height = attachmentViewportHeight(msg.Height)
	}
	if m.currentView == viewImage {
		return m, tea.Batch(clearKittyImagesCmd(), tea.ClearScreen)
	}

	if m.currentView == viewList {
		m.ensureCursorVisible()
	}
	return m, nil
}

// handleAttachmentsModalKey handles keyboard input in the attachments modal
func (m Model) handleAttachmentsModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	km := m.keyMap()
	switch {
	case key.Matches(msg, km.attachmentsModalKeys.Close):
		// Don't allow closing while downloading
		if m.attachments.modal.downloading {
			return m, nil
		}
		// Close attachments modal
		m.attachments.modal.show = false
		m.attachments.modal.attachments = nil
		m.attachments.modal.selectedIdx = 0
		return m, m.setWindowTitleCmd()
	case key.Matches(msg, km.attachmentsModalKeys.Down):
		// Don't allow navigation while downloading
		if m.attachments.modal.downloading {
			return m, nil
		}
		// Navigate down
		if m.attachments.modal.selectedIdx < len(m.attachments.modal.attachments)-1 {
			m.attachments.modal.selectedIdx++
		}
		return m, nil
	case key.Matches(msg, km.attachmentsModalKeys.Up):
		// Don't allow navigation while downloading
		if m.attachments.modal.downloading {
			return m, nil
		}
		// Navigate up
		if m.attachments.modal.selectedIdx > 0 {
			m.attachments.modal.selectedIdx--
		}
		return m, nil
	case key.Matches(msg, km.attachmentsModalKeys.Download):
		// Don't allow starting another download while one is in progress
		if m.attachments.modal.downloading || m.attachments.modal.loadingPreview {
			return m, nil
		}
		// Download selected attachment
		if m.attachments.modal.selectedIdx >= 0 &&
			m.attachments.modal.selectedIdx < len(m.attachments.modal.attachments) {
			m.attachments.modal.downloading = true
			return m, tea.Batch(
				m.downloadAttachmentCmd(m.attachments.modal.selectedIdx),
				m.ui.spinner.Tick,
			)
		}
		return m, nil
	case key.Matches(msg, km.attachmentsModalKeys.View):
		// View attachment inline (for images)
		if m.attachments.modal.downloading || m.attachments.modal.loadingPreview {
			return m, nil
		}
		if m.attachments.modal.selectedIdx >= 0 &&
			m.attachments.modal.selectedIdx < len(m.attachments.modal.attachments) {
			att := m.attachments.modal.attachments[m.attachments.modal.selectedIdx]
			if isImageMimeType(att.MimeType) || isTextAttachment(att.MimeType, att.Filename) {
				m.attachments.modal.loadingPreview = true
				return m, tea.Batch(
					m.loadAttachmentForViewCmd(m.attachments.modal.selectedIdx),
					m.ui.spinner.Tick,
				)
			}
			m.ui.err = fmt.Errorf("unsupported attachment type: %s", att.MimeType)
			m.ui.showError = true
			return m, nil
		}
		return m, nil
	}
	return m, nil
}

func availableMessageViewModes(msg gmail.Message) []messageViewMode {
	modes := make([]messageViewMode, 0, 3)
	if msg.BodyText != "" {
		modes = append(modes, viewModeText)
	}
	if msg.BodyHTML != "" {
		modes = append(modes, viewModeHTML)
	}
	modes = append(modes, viewModeRaw)
	return modes
}

func normalizeMessageViewMode(current messageViewMode, msg gmail.Message) messageViewMode {
	modes := availableMessageViewModes(msg)
	if slices.Contains(modes, current) {
		return current
	}
	return modes[0]
}

func nextMessageViewMode(current messageViewMode, msg gmail.Message) messageViewMode {
	modes := availableMessageViewModes(msg)
	if len(modes) == 1 {
		return modes[0]
	}
	current = normalizeMessageViewMode(current, msg)
	for i, mode := range modes {
		if mode == current {
			return modes[(i+1)%len(modes)]
		}
	}
	return modes[0]
}

func (m *Model) loadRawForExpandedMessages() tea.Cmd {
	if m.detail.currentThread == nil || len(m.detail.messages) == 0 {
		return nil
	}
	if m.detail.rawLoading == nil {
		m.detail.rawLoading = make(map[string]bool)
	}
	accountIndex := m.detail.currentThread.AccountIndex
	threadID := m.detail.currentThread.ThreadID

	var cmds []tea.Cmd
	for _, msg := range m.detail.messages {
		if !m.detail.expandedMessages[msg.ID] {
			continue
		}
		if msg.Raw != "" || m.detail.rawLoading[msg.ID] {
			continue
		}
		m.detail.rawLoading[msg.ID] = true
		cmds = append(cmds, m.loadMessageRawCmd(threadID, msg.ID, accountIndex))
	}

	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// reverseMessages reverses a slice of messages so newest appears first
func reverseMessages(messages []gmail.Message) []gmail.Message {
	reversed := make([]gmail.Message, len(messages))
	for i, msg := range messages {
		reversed[len(messages)-1-i] = msg
	}
	return reversed
}

func findThreadIndex(threads []gmail.Thread, accountIndex int, threadID string) int {
	for i := range threads {
		if threads[i].ThreadID == threadID && threads[i].AccountIndex == accountIndex {
			return i
		}
	}
	return -1
}
