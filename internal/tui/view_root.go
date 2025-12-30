package tui

import "strings"

// View renders the UI.
func (m Model) View() string {
	// Image/attachment views are special - full screen, no modals
	if m.currentView == viewImage {
		return m.renderImageView()
	}
	if m.currentView == viewAttachment {
		return m.renderAttachmentView()
	}

	var output string

	// If we just left image view, clear images and the screen
	if m.image.needsClear {
		var b strings.Builder
		b.WriteString(kittyClearCommands)
		b.WriteString("\x1b[2J") // Clear entire screen
		b.WriteString("\x1b[H")  // Move cursor to home

		// Render base view
		var baseView string
		if m.currentView == viewDetail {
			baseView = m.renderDetailView()
		} else {
			baseView = m.renderListView()
		}

		b.WriteString(baseView)
		output = b.String()
	} else {
		// Normal rendering without clearing
		if m.currentView == viewDetail {
			output = m.renderDetailView()
		} else {
			output = m.renderListView()
		}
	}

	// Overlay modals on top of base view
	switch {
	case m.ui.showHelp:
		output = m.overlayModal(output, m.renderHelpModal())
	case m.ui.showError && m.ui.err != nil:
		output = m.overlayModal(output, m.renderErrorModal())
	case m.inbox.delete.pending:
		output = m.overlayModal(output, m.renderDeleteModal())
	case m.attachments.modal.show:
		output = m.overlayModal(output, m.renderAttachmentsModal())
	}

	return m.ui.alert.Render(output)
}
