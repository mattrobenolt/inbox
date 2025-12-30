package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

// overlayModal centers a modal dialog on top of the base view.
func (m *Model) overlayModal(baseView string, modal string) string {
	// Simple modal box with border
	dialogBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2)

	// Render the styled modal
	styledModal := dialogBoxStyle.Render(modal)

	// Use overlay.Composite to place the modal on top of the base view
	return overlay.Composite(
		styledModal,    // foreground (modal)
		baseView,       // background (current view)
		overlay.Center, // horizontal position
		overlay.Center, // vertical position
		0,              // x offset
		0,              // y offset
	)
}

func (m *Model) renderHelpModal() string {
	var b strings.Builder

	modalWidth := max(40, min(80, m.ui.width-10))

	// Title - properly centered
	titleStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Align(lipgloss.Center).
		Bold(true)
	b.WriteString(titleStyle.Render("Keyboard Shortcuts"))
	b.WriteString("\n\n")

	helpModel := m.ui.help
	helpModel.ShowAll = true
	helpModel.Width = max(10, min(60, modalWidth-4))
	b.WriteString(helpModel.FullHelpView(splitHelpColumns(
		flattenHelpGroups(m.keyMap().FullHelp()),
		2,
	)))

	// Footer - properly centered
	b.WriteString("\n\n")
	footerStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color(m.theme.Modal.FooterFg))
	b.WriteString(footerStyle.Render("Press any key to close"))

	return b.String()
}

func flattenHelpGroups(groups [][]key.Binding) []key.Binding {
	var out []key.Binding
	for _, group := range groups {
		out = append(out, group...)
	}
	return out
}

func splitHelpColumns(bindings []key.Binding, columns int) [][]key.Binding {
	if columns <= 1 || len(bindings) == 0 {
		return [][]key.Binding{bindings}
	}
	perColumn := (len(bindings) + columns - 1) / columns
	out := make([][]key.Binding, 0, columns)
	for i := 0; i < len(bindings); i += perColumn {
		end := min(i+perColumn, len(bindings))
		out = append(out, bindings[i:end])
	}
	return out
}

func (m *Model) renderErrorModal() string {
	var b strings.Builder

	modalWidth := 60

	// Title - properly centered
	titleStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Align(lipgloss.Center)
	b.WriteString(titleStyle.Render("Error"))
	b.WriteString("\n\n")

	// Error message
	errorMsg := m.ui.err.Error()
	b.WriteString(errorMsg)
	b.WriteString("\n\n")

	// Footer - properly centered
	footerStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Align(lipgloss.Center)
	b.WriteString(footerStyle.Render("Press any key to dismiss"))

	return b.String()
}

func (m *Model) renderDeleteModal() string {
	var b strings.Builder

	modalWidth := 60
	titleStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Align(lipgloss.Center).
		Bold(true)
	title := "Trash Threads"
	switch m.inbox.delete.action {
	case deleteActionTrash:
		title = "Trash Threads"
	case deleteActionArchive:
		title = "Archive Threads"
	case deleteActionPermanent:
		title = "Delete Threads"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	count := len(m.inbox.delete.targets)
	if count <= 1 {
		switch m.inbox.delete.action {
		case deleteActionTrash:
			b.WriteString("Move this thread to trash?")
		case deleteActionArchive:
			b.WriteString("Archive this thread?")
		case deleteActionPermanent:
			b.WriteString("Permanently delete this thread? This cannot be undone.")
		}
		b.WriteString("\n")
		if count == 1 {
			if thread := m.threadForRef(m.inbox.delete.targets[0]); thread != nil {
				from := strings.TrimSpace(stripZeroWidth(thread.From))
				subject := strings.TrimSpace(stripZeroWidth(thread.Subject))
				maxWidth := modalWidth - 4
				if from != "" {
					b.WriteString("\nFrom: " + truncateToWidth(from, maxWidth))
				}
				if subject != "" {
					b.WriteString("\nSubject: " + truncateToWidth(subject, maxWidth))
				}
			}
		}
	} else {
		switch m.inbox.delete.action {
		case deleteActionTrash:
			b.WriteString(fmt.Sprintf("Move %d threads to trash?", count))
		case deleteActionArchive:
			b.WriteString(fmt.Sprintf("Archive %d threads?", count))
		case deleteActionPermanent:
			b.WriteString(fmt.Sprintf("Permanently delete %d threads? This cannot be undone.", count))
		}
	}

	b.WriteString("\n\n")
	footerStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color(m.theme.Modal.FooterFg))
	footer := "y confirm • n cancel"
	if m.inbox.delete.action == deleteActionPermanent {
		footer = "y delete • n cancel"
	}
	b.WriteString(footerStyle.Render(footer))

	return b.String()
}

func (m *Model) renderAttachmentsModal() string {
	var b strings.Builder

	modalWidth := 70

	// Title - properly centered
	titleStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Align(lipgloss.Center).
		Bold(true)

	title := fmt.Sprintf("%d Attachments", len(m.attachments.modal.attachments))
	if len(m.attachments.modal.attachments) == 1 {
		title = "1 Attachment"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// List each attachment
	for i, att := range m.attachments.modal.attachments {
		selected := i == m.attachments.modal.selectedIdx

		sizeStr := formatAttachmentSize(att.Size)

		// Build line
		var line string
		if selected {
			line = fmt.Sprintf("  > %s (%s)", att.Filename, sizeStr)
		} else {
			line = fmt.Sprintf("    %s (%s)", att.Filename, sizeStr)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	// Footer - properly centered
	b.WriteString("\n")
	footerStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color(m.theme.Modal.FooterFg))

	var footer string
	switch {
	case m.attachments.modal.downloading:
		footer = m.ui.spinner.View() + " Downloading..."
	case m.attachments.modal.loadingPreview:
		footer = m.ui.spinner.View() + " Loading preview..."
	default:
		footer = "j/k navigate • d download • v view • esc close"
		if m.attachments.modal.selectedIdx >= 0 &&
			m.attachments.modal.selectedIdx < len(m.attachments.modal.attachments) {
			selected := m.attachments.modal.attachments[m.attachments.modal.selectedIdx]
			if selected.Size > 0 {
				footer += " • " + formatAttachmentSize(selected.Size)
			}
		}
	}
	b.WriteString(footerStyle.Render(footer))

	return b.String()
}
