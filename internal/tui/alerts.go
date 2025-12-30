package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"go.dalton.dog/bubbleup"

	"go.withmatt.com/inbox/internal/config"
)

const toastDurationSeconds = 6

func newAlertModel(theme config.Theme, width int) bubbleup.AlertModel {
	model := *bubbleup.NewAlertModel(width, true, toastDurationSeconds)

	color := strings.TrimSpace(theme.Detail.BorderSelected)
	if color == "" {
		color = strings.TrimSpace(theme.Status.ModeBg)
	}
	if color == "" {
		color = theme.Status.Fg
	}

	model.RegisterNewAlertType(bubbleup.AlertDefinition{
		Key:       bubbleup.InfoKey,
		ForeColor: color,
		Prefix:    bubbleup.InfoNerdSymbol,
	})

	return model
}

func (m Model) updateAlerts(msg tea.Msg) (Model, tea.Cmd) {
	outAlert, alertCmd := m.ui.alert.Update(msg)
	m.ui.alert = outAlert.(bubbleup.AlertModel)
	return m, alertCmd
}

func (m *Model) undoToastCmd(action deleteAction, count int) tea.Cmd {
	if count <= 0 {
		return nil
	}

	verb := ""
	switch action {
	case deleteActionArchive:
		verb = "Archived"
	case deleteActionTrash:
		verb = "Trashed"
	case deleteActionPermanent:
		return nil
	}

	noun := "thread"
	if count != 1 {
		noun = "threads"
	}

	message := fmt.Sprintf("%s %d %s - u undo", verb, count, noun)
	return m.ui.alert.NewAlertCmd(bubbleup.InfoKey, message)
}

func (m Model) clearAlerts() Model {
	m.ui.alert = newAlertModel(m.theme, m.ui.width)
	return m
}
