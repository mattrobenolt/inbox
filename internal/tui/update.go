package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles events and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		return m.updateSpinner(msg)
	case tea.KeyMsg:
		return m.updateKey(msg)
	case tea.MouseMsg:
		return m.updateMouse(msg)
	case inboxLoadedMsg:
		return m.handleInboxLoaded(msg)
	case threadMetadataLoadedMsg:
		return m.handleThreadMetadataLoaded(msg)
	case batchLoadStartMsg:
		return m.handleBatchLoadStart(msg)
	case batchThreadMetadataLoadedMsg:
		return m.handleBatchThreadMetadataLoaded(msg)
	case threadLoadedMsg:
		return m.handleThreadLoaded(msg)
	case threadMarkedMsg:
		return m.handleThreadMarked(msg)
	case attachmentDownloadedMsg:
		return m.handleAttachmentDownloaded(msg)
	case clearImageFlagMsg:
		m.image.needsClear = false
		return m, nil
	case attachmentLoadedMsg:
		return m.handleAttachmentLoaded(msg)
	case searchDebounceMsg:
		return m.handleSearchDebounce(msg)
	case searchRemoteLoadedMsg:
		return m.handleSearchRemoteLoaded(msg)
	case autoRefreshMsg:
		return m.handleAutoRefresh(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	default:
		return m, nil
	}
}
