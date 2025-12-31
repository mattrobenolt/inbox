package tui

import "go.withmatt.com/inbox/internal/log"

func (m *Model) logf(format string, args ...any) {
	log.Printf(format, args...)
}
