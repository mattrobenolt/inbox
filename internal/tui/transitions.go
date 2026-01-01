package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"go.withmatt.com/inbox/internal/gmail"
)

func (m *Model) openThread(thread gmail.Thread) tea.Cmd {
	m.detail.currentThread = &thread
	m.currentView = viewDetail
	m.detail.loading = true

	var cmds []tea.Cmd
	cmds = append(cmds, m.setWindowTitleCmd())
	cmds = append(cmds, m.loadThreadCmd(m.detail.currentThread))
	if m.detail.currentThread.Unread {
		// Optimistically mark as read
		m.detail.currentThread.Unread = false
		for i := range m.inbox.threads {
			if m.inbox.threads[i].ThreadID == m.detail.currentThread.ThreadID {
				m.inbox.threads[i].Unread = false
				break
			}
		}
		// Send API request in background
		cmds = append(
			cmds,
			m.markThreadUnreadCmd(
				m.detail.currentThread.ThreadID,
				false,
				m.detail.currentThread.AccountIndex,
			),
		)
	}
	return tea.Batch(cmds...)
}

func (m *Model) exitDetailView() tea.Cmd {
	m.currentView = viewList
	m.resetDetail()
	return m.setWindowTitleCmd()
}

func (m *Model) showAttachmentsModal(msg gmail.Message) {
	m.attachments.modal.show = true
	m.attachments.modal.attachments = msg.Attachments
	m.attachments.modal.messageID = msg.ID
	if m.detail.currentThread != nil {
		m.attachments.modal.accountIndex = m.detail.currentThread.AccountIndex
	}
	m.attachments.modal.selectedIdx = 0
}

func (m *Model) enterAttachmentView(msg attachmentLoadedMsg) error {
	decoded, err := decodeAttachmentData(msg.data)
	if err != nil {
		return fmt.Errorf("failed to decode %s: %w", msg.filename, err)
	}

	raw := string(decoded)
	rendered := m.renderMarkdown(raw, m.ui.width)
	if isHTMLAttachment(msg.mimeType, msg.filename) {
		cleanedHTML := cleanHTMLForConversion(raw)
		markdown, err := m.renderers.htmlConverter.ConvertString(cleanedHTML)
		if err == nil {
			rendered = m.renderMarkdown(markdown, m.ui.width)
		}
	}

	m.attachments.preview.filename = msg.filename
	m.attachments.preview.mimeType = msg.mimeType
	m.attachments.preview.raw = raw
	m.attachments.preview.rendered = rendered
	m.attachments.preview.size = msg.size
	m.detail.savedViewportYOffset = m.detail.viewport.YOffset
	m.currentView = viewAttachment

	m.detail.viewport.Width = m.ui.width
	m.detail.viewport.Height = attachmentViewportHeight(m.ui.height)
	m.detail.viewport.SetContent(rendered)
	m.detail.viewport.GotoTop()
	m.detail.viewport.YOffset = 0

	return nil
}

func (m *Model) exitAttachmentView() tea.Cmd {
	m.currentView = viewDetail
	m.resetAttachmentPreview()
	m.detail.viewport.Width = m.ui.width
	m.detail.viewport.Height = detailViewportHeight(m.ui.height)
	body := m.renderThreadBody()
	m.detail.viewport.SetContent(body)
	m.detail.viewport.SetYOffset(m.detail.savedViewportYOffset)
	return tea.Batch(tea.ClearScreen, m.setWindowTitleCmd())
}

func (m *Model) enterImageView(msg attachmentLoadedMsg) {
	m.image.data = msg.data
	m.image.mimeType = msg.mimeType
	m.image.filename = msg.filename
	m.image.size = msg.size
	m.currentView = viewImage
}

func (m *Model) exitImageView() tea.Cmd {
	m.currentView = viewDetail
	m.image.data = ""
	m.image.mimeType = ""
	m.image.filename = ""
	m.image.size = 0
	m.image.needsClear = true

	clearFlagCmd := func() tea.Msg {
		return clearImageFlagMsg{}
	}
	return tea.Batch(clearKittyImagesCmd(), clearFlagCmd, m.setWindowTitleCmd())
}
