package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	windowTitleMaxRunes = 80
	windowTitleSuffix   = " - inbox"
)

func (m *Model) setWindowTitleCmd() tea.Cmd {
	return tea.SetWindowTitle(m.windowTitle())
}

func (m *Model) windowTitle() string {
	if m.attachments.modal.show {
		return formatWindowTitle("Attachments")
	}

	switch m.currentView {
	case viewList:
		return formatWindowTitle("")
	case viewDetail:
		return formatWindowTitle(m.detailTitle())
	case viewAttachment:
		if name := strings.TrimSpace(m.attachments.preview.filename); name != "" {
			return formatWindowTitle("Attachment: " + stripZeroWidth(name))
		}
		return formatWindowTitle("Attachment")
	case viewImage:
		if name := strings.TrimSpace(m.image.filename); name != "" {
			return formatWindowTitle("Image: " + stripZeroWidth(name))
		}
		return formatWindowTitle("Image")
	default:
		return formatWindowTitle("")
	}
}

func (m *Model) detailTitle() string {
	if m.detail.currentThread == nil {
		return "Message"
	}
	subject := strings.TrimSpace(stripZeroWidth(m.detail.currentThread.Subject))
	if subject != "" {
		return subject
	}
	from := strings.TrimSpace(stripZeroWidth(m.detail.currentThread.From))
	if from != "" {
		return "Message from " + from
	}
	return "Message"
}

func formatWindowTitle(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return "inbox"
	}
	maxBody := windowTitleMaxRunes - len([]rune(windowTitleSuffix))
	if maxBody <= 0 {
		return "inbox"
	}
	body = truncateTitle(body, maxBody)
	return body + windowTitleSuffix
}

func truncateTitle(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	if maxRunes <= 3 {
		return strings.Repeat(".", maxRunes)
	}
	return string(runes[:maxRunes-3]) + "..."
}
