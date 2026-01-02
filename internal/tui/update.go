package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles events and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var model tea.Model

	switch msg := msg.(type) {
	case spinner.TickMsg:
		model, cmd = m.updateSpinner(msg)
	case tea.KeyMsg:
		model, cmd = m.updateKey(msg)
	case tea.MouseMsg:
		model, cmd = m.updateMouse(msg)
	case tea.FocusMsg:
		m.ui.focused = true
		model = m
	case tea.BlurMsg:
		m.ui.focused = false
		model = m
	case inboxLoadedMsg:
		model, cmd = m.handleInboxLoaded(msg)
	case threadMetadataLoadedMsg:
		model = m.handleThreadMetadataLoaded(msg)
	case batchLoadStartMsg:
		model = m.handleBatchLoadStart(msg)
	case batchThreadMetadataLoadedMsg:
		model = m.handleBatchThreadMetadataLoaded(msg)
	case threadLoadedMsg:
		model, cmd = m.handleThreadLoaded(msg)
	case threadMarkedMsg:
		model = m.handleThreadMarked(msg)
	case threadsActionMsg:
		model, cmd = m.handleThreadsAction(msg)
	case threadsUndoMsg:
		model = m.handleThreadsUndo(msg)
	case attachmentDownloadedMsg:
		model = m.handleAttachmentDownloaded(msg)
	case clearImageFlagMsg:
		m.image.needsClear = false
		model = m
	case attachmentLoadedMsg:
		model, cmd = m.handleAttachmentLoaded(msg)
	case messageRawLoadedMsg:
		model = m.handleMessageRawLoaded(msg)
	case searchDebounceMsg:
		model, cmd = m.handleSearchDebounce(msg)
	case searchRemoteLoadedMsg:
		model, cmd = m.handleSearchRemoteLoaded(msg)
	case autoRefreshMsg:
		model, cmd = m.handleAutoRefresh()
	case linkScanFinishedMsg:
		model = m.handleLinkScanFinished(msg)
	case tea.WindowSizeMsg:
		model, cmd = m.handleWindowSize(msg)
	default:
		model = m
	}

	if model == nil {
		model = m
	}

	updatedModel, ok := model.(Model)
	if !ok {
		return model, cmd
	}
	updatedModel, alertCmd := updatedModel.updateAlerts(msg)
	return updatedModel, tea.Batch(cmd, alertCmd)
}
