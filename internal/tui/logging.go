package tui

import "log"

func (m *Model) logf(format string, args ...any) {
	if !m.debug {
		return
	}
	log.Printf("DEBUG: "+format, args...)
}
