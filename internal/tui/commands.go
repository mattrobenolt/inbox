package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/sync/errgroup"

	"go.withmatt.com/inbox/internal/gmail"
)

const (
	// threadLoadConcurrency controls how many threads we fetch in parallel
	threadLoadConcurrency = 5
)

type threadLoadedMsg struct {
	messages []gmail.Message
	err      error
}

type inboxLoadSource int

const (
	inboxLoadInit inboxLoadSource = iota
	inboxLoadManual
	inboxLoadAuto
	inboxLoadPage
)

type inboxLoadedMsg struct {
	threads       []gmail.Thread
	nextPageToken string
	append        bool // If true, append to existing threads instead of replacing
	source        inboxLoadSource
	err           error
}

type threadMetadataLoadedMsg struct {
	index        int
	threadID     string
	accountIndex int
	thread       *gmail.Thread
	err          error
}

type batchThreadMetadataLoadedMsg struct {
	results []threadMetadataLoadedMsg
}

type batchLoadStartMsg struct {
	count int // Number of threads starting to load
}

type threadMarkedMsg struct {
	threadID string
	unread   bool
	err      error
}

type threadsActionMsg struct {
	action deleteAction
	refs   []threadRef
	failed []threadRef
	err    error
}

type threadsUndoMsg struct {
	action deleteAction
	refs   []threadRef
	failed []threadRef
	err    error
}

type clearImageFlagMsg struct{}

type searchDebounceMsg struct {
	query      string
	generation int
}

type autoRefreshMsg struct{}

type searchRemoteLoadedMsg struct {
	query      string
	generation int
	threads    []gmail.Thread
	err        error
}

// loadInboxCmd loads the inbox asynchronously from all accounts and merges them
func (m *Model) loadInboxCmd(source inboxLoadSource) tea.Cmd {
	return func() tea.Msg {
		m.logf("LoadInbox start accounts=%d", len(m.clients))
		if len(m.clients) == 0 {
			return inboxLoadedMsg{
				threads: nil,
				source:  source,
				err:     errors.New("no accounts configured"),
			}
		}

		// Load from all accounts in parallel
		g, ctx := errgroup.WithContext(m.ctx)

		// Results from each account
		type accountResult struct {
			accountIndex int
			threads      []gmail.Thread
			pageToken    string
			err          error
		}

		results := make([]accountResult, len(m.clients))

		for i := range m.clients {
			accountIndex := i
			g.Go(func() error {
				m.logf("ListInbox start account=%d", accountIndex)
				inbox, err := m.clients[accountIndex].ListInbox(ctx, 50, "")
				if err != nil {
					m.logf("ListInbox error account=%d err=%v", accountIndex, err)
					results[accountIndex] = accountResult{
						accountIndex: accountIndex,
						threads:      nil,
						pageToken:    "",
						err:          err,
					}
					return nil // Don't fail entire group on single account error
				}
				results[accountIndex] = accountResult{
					accountIndex: accountIndex,
					threads:      inbox.Threads,
					pageToken:    inbox.NextPageToken,
					err:          nil,
				}
				m.logf(
					"ListInbox done account=%d threads=%d next=%s",
					accountIndex,
					len(inbox.Threads),
					inbox.NextPageToken,
				)
				return nil
			})
		}

		// Wait for all accounts to finish
		g.Wait()

		// Check for errors
		for _, result := range results {
			if result.err != nil {
				return inboxLoadedMsg{threads: nil, source: source, err: result.err}
			}
		}

		// Merge threads from all accounts and tag with account info
		var allThreads []gmail.Thread
		for _, result := range results {
			m.logf("Merge inbox account=%d threads=%d", result.accountIndex, len(result.threads))
			// Tag each thread with account info
			for i := range result.threads {
				result.threads[i].AccountIndex = result.accountIndex
				result.threads[i].AccountName = m.accountNames[result.accountIndex]
			}
			allThreads = append(allThreads, result.threads...)
		}

		// Don't sort now - threads have no dates yet (just IDs)
		// They'll get sorted automatically as metadata loads in background
		// (see batchThreadMetadataLoadedMsg handler which sorts after each batch)

		// TODO: Handle pagination for multiple accounts
		// For now, just return empty pageToken since pagination is complex with multiple accounts
		return inboxLoadedMsg{
			threads:       allThreads,
			nextPageToken: "",
			append:        false,
			source:        source,
			err:           nil,
		}
	}
}

func (m *Model) autoRefreshCmd() tea.Cmd {
	if m.uiConfig.RefreshIntervalSeconds <= 0 {
		return nil
	}
	interval := time.Duration(m.uiConfig.RefreshIntervalSeconds) * time.Second
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return autoRefreshMsg{}
	})
}

func (m *Model) searchDebounceCmd(query string, generation int) tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
		return searchDebounceMsg{query: query, generation: generation}
	})
}

func (m *Model) searchRemoteCmd(query string, generation int) tea.Cmd {
	return func() tea.Msg {
		if len(m.clients) == 0 {
			return searchRemoteLoadedMsg{
				query:      query,
				generation: generation,
				err:        errors.New("no accounts configured"),
			}
		}

		g, ctx := errgroup.WithContext(m.ctx)

		type accountResult struct {
			accountIndex int
			threads      []gmail.Thread
			err          error
		}

		results := make([]accountResult, len(m.clients))

		for i := range m.clients {
			accountIndex := i
			g.Go(func() error {
				m.logf("SearchInbox start account=%d query=%q", accountIndex, query)
				inbox, err := m.clients[accountIndex].SearchInbox(ctx, query, 50, "")
				if err != nil {
					m.logf("SearchInbox error account=%d query=%q err=%v", accountIndex, query, err)
					results[accountIndex] = accountResult{accountIndex: accountIndex, err: err}
					return nil
				}
				results[accountIndex] = accountResult{
					accountIndex: accountIndex,
					threads:      inbox.Threads,
				}
				m.logf(
					"SearchInbox done account=%d query=%q threads=%d",
					accountIndex,
					query,
					len(inbox.Threads),
				)
				return nil
			})
		}

		g.Wait()

		var allThreads []gmail.Thread
		for _, result := range results {
			if result.err != nil {
				return searchRemoteLoadedMsg{query: query, generation: generation, err: result.err}
			}
			for i := range result.threads {
				result.threads[i].AccountIndex = result.accountIndex
				result.threads[i].AccountName = m.accountNames[result.accountIndex]
			}
			allThreads = append(allThreads, result.threads...)
		}

		return searchRemoteLoadedMsg{query: query, generation: generation, threads: allThreads}
	}
}

func (m *Model) loadThreadsMetadataCmd(indices []int) tea.Cmd {
	if len(indices) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(indices))
	for _, idx := range indices {
		if idx < 0 || idx >= len(m.inbox.threads) {
			continue
		}
		thread := m.inbox.threads[idx]
		cmds = append(cmds, m.loadSingleThreadMetadataCmd(loadRequest{
			index:        idx,
			threadID:     thread.ThreadID,
			accountIndex: thread.AccountIndex,
		}))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// loadMoreThreadsCmd loads the next page of threads
// TODO: Handle pagination for multiple accounts
func (m *Model) loadMoreThreadsCmd() tea.Cmd {
	if m.inbox.nextPageToken == "" {
		return nil
	}

	pageToken := m.inbox.nextPageToken
	return func() tea.Msg {
		// For now, just load from the first account
		inbox, err := m.clients[0].ListInbox(m.ctx, 50, pageToken)
		if err != nil {
			return inboxLoadedMsg{threads: nil, err: err, append: true, source: inboxLoadPage}
		}
		return inboxLoadedMsg{
			threads:       inbox.Threads,
			nextPageToken: inbox.NextPageToken,
			append:        true,
			source:        inboxLoadPage,
			err:           nil,
		}
	}
}

// loadThreadCmd loads a thread asynchronously from Gmail API
func (m *Model) loadThreadCmd(thread *gmail.Thread) tea.Cmd {
	return func() tea.Msg {
		if thread != nil {
			m.logf("GetThread start account=%d thread=%s", thread.AccountIndex, thread.ThreadID)
		}
		messages, err := m.clients[thread.AccountIndex].GetThread(m.ctx, thread.ThreadID)
		if thread != nil {
			m.logf(
				"GetThread done account=%d thread=%s messages=%d err=%v",
				thread.AccountIndex,
				thread.ThreadID,
				len(messages),
				err,
			)
		}
		return threadLoadedMsg{
			messages: messages,
			err:      err,
		}
	}
}

// loadVisibleThreadsCmd loads metadata for all visible threads that aren't loaded yet
// Uses errgroup to load threads in parallel with controlled concurrency
func (m *Model) loadVisibleThreadsCmd() tea.Cmd {
	return m.loadVisibleThreadsCmdWithForce(false)
}

type loadRequest struct {
	index        int
	threadID     string
	accountIndex int
}

// loadAllThreadsMetadataCmd loads metadata for ALL threads, streaming results as they arrive
func (m *Model) loadAllThreadsMetadataCmd(force bool) tea.Cmd {
	// Collect all threads that need loading
	var toLoad []loadRequest
	for i := 0; i < len(m.inbox.threads); i++ {
		if !m.inbox.threads[i].Loaded || force {
			toLoad = append(toLoad, loadRequest{
				index:        i,
				threadID:     m.inbox.threads[i].ThreadID,
				accountIndex: m.inbox.threads[i].AccountIndex,
			})
		}
	}

	if len(toLoad) == 0 {
		return nil
	}

	// Create individual commands for each thread - they'll stream back as they complete
	cmds := make([]tea.Cmd, 0, len(toLoad))
	for _, req := range toLoad {
		cmds = append(cmds, m.loadSingleThreadMetadataCmd(req))
	}

	return tea.Batch(cmds...)
}

// loadSingleThreadMetadataCmd loads metadata for a single thread
func (m *Model) loadSingleThreadMetadataCmd(req loadRequest) tea.Cmd {
	return func() tea.Msg {
		thread, err := m.clients[req.accountIndex].GetThreadMetadata(m.ctx, req.threadID)
		return threadMetadataLoadedMsg{
			index:        req.index,
			threadID:     req.threadID,
			accountIndex: req.accountIndex,
			thread:       thread,
			err:          err,
		}
	}
}

// loadVisibleThreadsCmdWithForce loads metadata for visible threads, optionally forcing reload even if already loaded
func (m *Model) loadVisibleThreadsCmdWithForce(force bool) tea.Cmd {
	// Collect threads that need loading
	var toLoad []loadRequest
	for _, idx := range m.visibleThreadIndices() {
		if idx < 0 || idx >= len(m.inbox.threads) {
			continue
		}
		// Load if not loaded, or if force is true
		if !m.inbox.threads[idx].Loaded || force {
			toLoad = append(toLoad, loadRequest{
				index:        idx,
				threadID:     m.inbox.threads[idx].ThreadID,
				accountIndex: m.inbox.threads[idx].AccountIndex,
			})
		}
	}

	if len(toLoad) == 0 {
		return nil
	}

	// Create a batch command that sends start message and then loads
	return tea.Batch(
		func() tea.Msg {
			return batchLoadStartMsg{count: len(toLoad)}
		},
		func() tea.Msg {
			g, ctx := errgroup.WithContext(m.ctx)

			// Limit concurrency
			g.SetLimit(threadLoadConcurrency)

			var mu sync.Mutex
			results := make([]threadMetadataLoadedMsg, 0, len(toLoad))

			for _, req := range toLoad {
				request := req // Capture for closure
				g.Go(func() error {
					// Use the correct client for this thread's account
					thread, err := m.clients[request.accountIndex].GetThreadMetadata(
						ctx,
						request.threadID,
					)

					mu.Lock()
					defer mu.Unlock()
					results = append(results, threadMetadataLoadedMsg{
						index:        request.index,
						threadID:     request.threadID,
						accountIndex: request.accountIndex,
						thread:       thread,
						err:          err,
					})

					// Don't return error to errgroup - we handle per-thread errors
					return nil
				})
			}

			// Wait for all to complete
			g.Wait()

			return batchThreadMetadataLoadedMsg{results: results}
		},
	)
}

// markThreadUnreadCmd marks a thread with the specified unread state
func (m *Model) markThreadUnreadCmd(threadID string, unread bool, accountIndex int) tea.Cmd {
	return func() tea.Msg {
		var err error

		if unread {
			err = m.clients[accountIndex].MarkThreadUnread(m.ctx, threadID)
		} else {
			err = m.clients[accountIndex].MarkThreadRead(m.ctx, threadID)
		}

		return threadMarkedMsg{
			threadID: threadID,
			unread:   unread,
			err:      err,
		}
	}
}

// threadActionCmd performs an action on the specified threads.
func (m *Model) threadActionCmd(action deleteAction, refs []threadRef) tea.Cmd {
	return func() tea.Msg {
		if len(refs) == 0 {
			return threadsActionMsg{action: action}
		}

		failed := make([]threadRef, 0)
		var firstErr error
		for _, ref := range refs {
			if ref.accountIndex < 0 || ref.accountIndex >= len(m.clients) {
				if firstErr == nil {
					firstErr = fmt.Errorf("invalid account index %d", ref.accountIndex)
				}
				failed = append(failed, ref)
				continue
			}
			var err error
			switch action {
			case deleteActionTrash:
				err = m.clients[ref.accountIndex].TrashThread(m.ctx, ref.threadID)
			case deleteActionArchive:
				err = m.clients[ref.accountIndex].ArchiveThread(m.ctx, ref.threadID)
			case deleteActionPermanent:
				err = m.clients[ref.accountIndex].DeleteThread(m.ctx, ref.threadID)
			}
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				failed = append(failed, ref)
			}
		}

		var err error
		if len(failed) > 0 {
			err = fmt.Errorf("failed to delete %d thread(s): %w", len(failed), firstErr)
		}

		return threadsActionMsg{
			action: action,
			refs:   refs,
			failed: failed,
			err:    err,
		}
	}
}

// undoThreadsCmd reverses the last archive or trash action.
func (m *Model) undoThreadsCmd(action deleteAction, refs []threadRef) tea.Cmd {
	return func() tea.Msg {
		if len(refs) == 0 {
			return threadsUndoMsg{action: action}
		}

		failed := make([]threadRef, 0)
		var firstErr error
		for _, ref := range refs {
			if ref.accountIndex < 0 || ref.accountIndex >= len(m.clients) {
				if firstErr == nil {
					firstErr = fmt.Errorf("invalid account index %d", ref.accountIndex)
				}
				failed = append(failed, ref)
				continue
			}
			var err error
			switch action {
			case deleteActionArchive:
				err = m.clients[ref.accountIndex].UnarchiveThread(m.ctx, ref.threadID)
			case deleteActionTrash:
				err = m.clients[ref.accountIndex].UntrashThread(m.ctx, ref.threadID)
			case deleteActionPermanent:
				err = errors.New("cannot undo permanent delete")
			}
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				failed = append(failed, ref)
			}
		}

		var err error
		if len(failed) > 0 {
			err = fmt.Errorf("failed to undo %d thread(s): %w", len(failed), firstErr)
		}

		return threadsUndoMsg{
			action: action,
			refs:   refs,
			failed: failed,
			err:    err,
		}
	}
}

type attachmentDownloadedMsg struct {
	filename string
	err      error
}

type attachmentLoadedMsg struct {
	data     string // base64url data from Gmail API
	mimeType string
	filename string
	size     int64
	err      error
}

// downloadAttachmentCmd downloads an attachment to ~/Downloads
func (m *Model) downloadAttachmentCmd(attachmentIdx int) tea.Cmd {
	return func() tea.Msg {
		if attachmentIdx < 0 || attachmentIdx >= len(m.attachments.modal.attachments) {
			return attachmentDownloadedMsg{err: errors.New("invalid attachment index")}
		}

		att := m.attachments.modal.attachments[attachmentIdx]

		// Get home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return attachmentDownloadedMsg{filename: att.Filename, err: err}
		}

		// Create Downloads directory if it doesn't exist
		downloadsDir := filepath.Join(homeDir, "Downloads")
		if err := os.MkdirAll(downloadsDir, 0o750); err != nil {
			return attachmentDownloadedMsg{filename: att.Filename, err: err}
		}

		// Create file
		filePath := filepath.Join(downloadsDir, att.Filename)
		file, err := os.Create(filePath)
		if err != nil {
			return attachmentDownloadedMsg{filename: att.Filename, err: err}
		}
		defer file.Close()

		// Stream download and decode directly to file
		err = m.clients[m.attachments.modal.accountIndex].DownloadAttachmentToWriter(
			m.ctx,
			m.attachments.modal.messageID,
			att.AttachmentID,
			file,
		)
		if err != nil {
			// Clean up partial file on error
			os.Remove(filePath)
			return attachmentDownloadedMsg{filename: att.Filename, err: err}
		}

		return attachmentDownloadedMsg{filename: att.Filename, err: nil}
	}
}

// loadAttachmentForViewCmd loads attachment data for inline viewing
func (m *Model) loadAttachmentForViewCmd(attachmentIdx int) tea.Cmd {
	return func() tea.Msg {
		if attachmentIdx < 0 || attachmentIdx >= len(m.attachments.modal.attachments) {
			return attachmentLoadedMsg{err: errors.New("invalid attachment index")}
		}

		att := m.attachments.modal.attachments[attachmentIdx]

		// Get raw base64url attachment data from the correct account
		data, err := m.clients[m.attachments.modal.accountIndex].GetAttachmentData(
			m.ctx,
			m.attachments.modal.messageID,
			att.AttachmentID,
		)
		if err != nil {
			return attachmentLoadedMsg{filename: att.Filename, err: err}
		}

		return attachmentLoadedMsg{
			data:     data,
			mimeType: att.MimeType,
			filename: att.Filename,
			size:     att.Size,
			err:      nil,
		}
	}
}

func clearKittyImagesCmd() tea.Cmd {
	return func() tea.Msg {
		fmt.Print(kittyClearCommands)
		return nil
	}
}

func bellCmd() tea.Cmd {
	return func() tea.Msg {
		fmt.Print("\a")
		return nil
	}
}

// sortThreadsByDate sorts threads by date (newest first)
// Handles zero dates (unloaded threads) by keeping them at the end
func sortThreadsByDate(threads []gmail.Thread) {
	// Sort newest first. Threads without dates (unloaded) stay at the end.
	sort.SliceStable(threads, func(i, j int) bool {
		di := threads[i].Date
		dj := threads[j].Date
		if di.IsZero() && dj.IsZero() {
			return false
		}
		if di.IsZero() {
			return false
		}
		if dj.IsZero() {
			return true
		}
		return di.After(dj)
	})
}
