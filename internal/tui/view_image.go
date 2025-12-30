package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"go.withmatt.com/inbox/internal/image"
)

func (m *Model) renderImageView() string {
	var b strings.Builder

	// Create transformer and get dimensions
	transformer, imgWidth, imgHeight, err := image.NewImageTransformer(
		m.image.data,
		m.image.mimeType,
	)
	if err != nil {
		// Error creating transformer - show error and instructions
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Image.ErrorFg))
		return fmt.Sprintf(
			"\n\n%s\n\nPress ESC to go back",
			errorStyle.Render(fmt.Sprintf("Error loading image: %v", err)),
		)
	}
	defer transformer.Close()

	// Leave margins and reserve the statusline row.
	maxCols := m.ui.width - 4
	maxRows := m.ui.height - 2
	if maxCols < 1 {
		maxCols = 1
	}
	if maxRows < 1 {
		maxRows = 1
	}

	pixelsPerColumn, pixelsPerRow := terminalPixelsPerCell()

	dstCols := maxCols
	dstRows := maxRows
	if imgWidth > 0 && imgHeight > 0 {
		imgRatio := float64(imgWidth) / float64(imgHeight)
		termRatio := float64(maxCols*pixelsPerColumn) / float64(maxRows*pixelsPerRow)
		if imgRatio > termRatio {
			dstCols = maxCols
			dstRows = int(float64(dstCols*pixelsPerColumn) / imgRatio / float64(pixelsPerRow))
		} else {
			dstRows = maxRows
			dstCols = int(float64(dstRows*pixelsPerRow) * imgRatio / float64(pixelsPerColumn))
		}
	}
	if dstCols < 1 {
		dstCols = 1
	}
	if dstRows < 1 {
		dstRows = 1
	}
	if dstCols > maxCols {
		dstCols = maxCols
	}
	if dstRows > maxRows {
		dstRows = maxRows
	}
	constraintParam := fmt.Sprintf("c=%d,r=%d", dstCols, dstRows)

	// Clear the screen with blank lines to make a clean canvas
	// Fill the entire terminal height with newlines
	b.WriteString(kittyClearCommands)
	for i := 0; i < m.ui.height; i++ {
		b.WriteString("\n")
	}

	// Move cursor back to top
	b.WriteString("\x1b[H")

	// Render image with Kitty protocol
	// f=100 is PNG format, a=T is transmit and display
	fmt.Fprintf(&b, "\x1b_Gf=100,a=T,t=d,%s;", constraintParam)

	// Stream image data
	io.Copy(&b, transformer)

	// End Kitty escape sequence
	b.WriteString("\x1b\\")

	left := []statusSegment{statusModeSegment(m.theme, "ATTACH")}
	right := []statusSegment{statusDimSegment(m.theme, "esc/q back")}

	infoParts := make([]string, 0, 2)
	if m.image.filename != "" {
		infoParts = append(infoParts, m.image.filename)
	}
	if m.image.mimeType != "" {
		infoParts = append(infoParts, m.image.mimeType)
	}
	if m.image.size > 0 {
		infoParts = append(infoParts, formatAttachmentSize(m.image.size))
	}
	info := strings.Join(infoParts, " • ")
	if info != "" {
		maxWidth := 0
		if m.ui.width > 0 {
			maxWidth = m.ui.width -
				lipgloss.Width(renderStatusSegments(left)) -
				lipgloss.Width(renderStatusSegments(right)) - 1
		}
		if maxWidth > 0 {
			info = truncateToWidth(info, maxWidth)
		}
		if info != "" {
			left = append(left, statusDimSegment(m.theme, info))
		}
	}

	footer := renderStatusline(m.theme, m.ui.width, left, right)
	fmt.Fprintf(&b, "\x1b[%d;1H", max(1, m.ui.height))
	b.WriteString(footer)

	return b.String()
}
