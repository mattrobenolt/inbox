package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"go.withmatt.com/inbox/internal/config"
)

type statusSegment struct {
	text  string
	style lipgloss.Style
	raw   bool
}

const statusSeparatorGlyph = ""

func statusBaseStyle(theme config.Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(theme.Status.Bg)).
		Foreground(lipgloss.Color(theme.Status.Fg))
}

func statusSegmentStyle(theme config.Theme) lipgloss.Style {
	return statusBaseStyle(theme).Padding(0, 1)
}

func statusDimStyle(theme config.Theme) lipgloss.Style {
	return statusBaseStyle(theme).
		Foreground(lipgloss.Color(theme.Status.Dim)).
		Padding(0, 1)
}

func statusModeStyle(theme config.Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(theme.Status.ModeBg)).
		Foreground(lipgloss.Color(theme.Status.ModeFg)).
		Bold(true).
		Padding(0, 1)
}

func statusTabStyle(theme config.Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(theme.Status.TabBg)).
		Foreground(lipgloss.Color(theme.Status.TabFg)).
		Bold(true).
		Padding(0, 1)
}

func statusTextSegment(theme config.Theme, text string) statusSegment {
	return statusSegment{text: text, style: statusSegmentStyle(theme)}
}

func statusDimSegment(theme config.Theme, text string) statusSegment {
	return statusSegment{text: text, style: statusDimStyle(theme)}
}

func statusModeSegment(theme config.Theme, text string) statusSegment {
	return statusSegment{text: text, style: statusModeStyle(theme)}
}

func statusTabSegment(theme config.Theme, text string) statusSegment {
	return statusSegment{text: text, style: statusTabStyle(theme)}
}

func statusPowerlineSeparator(leftBg string, rightBg string) statusSegment {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color(rightBg)).
		Foreground(lipgloss.Color(leftBg))
	return statusSegment{text: statusSeparatorGlyph, style: style}
}

func statusRawSegment(text string) statusSegment {
	return statusSegment{text: text, raw: true}
}

func statusPaddedRaw(theme config.Theme, text string) statusSegment {
	pad := statusBaseStyle(theme).Render(" ")
	return statusRawSegment(pad + text + pad)
}

func renderStatusline(
	theme config.Theme,
	width int,
	left []statusSegment,
	right []statusSegment,
) string {
	leftLine := renderStatusSegments(left)
	rightLine := renderStatusSegments(right)

	if width <= 0 {
		if rightLine == "" {
			return leftLine
		}
		if leftLine == "" {
			return rightLine
		}
		return leftLine + statusBaseStyle(theme).Render(" ") + rightLine
	}

	gap := max(width-lipgloss.Width(leftLine)-lipgloss.Width(rightLine), 1)
	filler := statusBaseStyle(theme).Render(strings.Repeat(" ", gap))
	return leftLine + filler + rightLine
}

func renderStatusSegments(segments []statusSegment) string {
	var b strings.Builder
	for _, seg := range segments {
		if seg.text == "" {
			continue
		}
		if seg.raw {
			b.WriteString(seg.text)
			continue
		}
		b.WriteString(seg.style.Render(seg.text))
	}
	return b.String()
}

func truncateToWidth(text string, maxWidth int) string {
	if maxWidth <= 0 || text == "" {
		return ""
	}
	if lipgloss.Width(text) <= maxWidth {
		return text
	}
	if maxWidth <= 3 {
		return strings.Repeat(".", maxWidth)
	}

	targetWidth := maxWidth - 3
	var b strings.Builder
	width := 0
	for _, r := range text {
		runeWidth := lipgloss.Width(string(r))
		if width+runeWidth > targetWidth {
			break
		}
		b.WriteRune(r)
		width += runeWidth
	}
	return b.String() + "..."
}
