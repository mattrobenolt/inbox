package log

import (
	stdlog "log"
	"os"

	"github.com/adrg/xdg"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	debugEnabled bool
	logFile      *os.File
)

func Setup(debug bool) error {
	debugEnabled = debug
	if !debug || logFile != nil {
		return nil
	}
	logPath, err := xdg.StateFile("inbox/debug.log")
	if err != nil {
		return err
	}
	logFile, err = tea.LogToFile(logPath, "inbox")
	return err
}

func Close() error {
	if logFile == nil {
		return nil
	}
	defer func() { logFile = nil }()
	return logFile.Close()
}

func DebugEnabled() bool {
	return debugEnabled
}

func Printf(format string, args ...any) {
	if debugEnabled {
		stdlog.Printf("DEBUG: "+format, args...)
	}
}
