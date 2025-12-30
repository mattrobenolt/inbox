package tui

import (
	"os"

	"golang.org/x/sys/unix"
)

const (
	defaultPixelsPerColumn = 10
	defaultPixelsPerRow    = 20
	maxPixelsPerCell       = 100
	kittyClearCommands     = "\x1b_Ga=d\x1b\\\x1b_Ga=d,p=1\x1b\\\x1b[0m"
)

func terminalPixelsPerCell() (int, int) {
	fds := []int{int(os.Stdout.Fd()), int(os.Stdin.Fd()), int(os.Stderr.Fd())}
	for _, fd := range fds {
		ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
		if err != nil {
			continue
		}
		if ws.Col == 0 || ws.Row == 0 || ws.Xpixel == 0 || ws.Ypixel == 0 {
			continue
		}

		ppc := int(ws.Xpixel) / int(ws.Col)
		ppr := int(ws.Ypixel) / int(ws.Row)
		if ppc > 0 && ppr > 0 && ppc < maxPixelsPerCell && ppr < maxPixelsPerCell {
			return ppc, ppr
		}
	}

	return defaultPixelsPerColumn, defaultPixelsPerRow
}
