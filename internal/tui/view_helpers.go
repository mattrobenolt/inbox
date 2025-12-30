package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Helper to format relative time.
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

// cleanHTMLForConversion preprocesses HTML to remove problematic elements.
func cleanHTMLForConversion(html string) string {
	// Remove style and script tags entirely
	styleRe := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	html = styleRe.ReplaceAllString(html, "")
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	html = scriptRe.ReplaceAllString(html, "")

	// Convert tables to simpler structure - just extract text content
	// This is a hack but marketing emails use tables for layout, not data
	tableRe := regexp.MustCompile(`(?is)<table[^>]*>.*?</table>`)
	html = tableRe.ReplaceAllStringFunc(html, func(table string) string {
		// Remove table structure tags but keep content
		cleaned := strings.ReplaceAll(table, "<tr>", "\n")
		cleaned = strings.ReplaceAll(cleaned, "</tr>", "\n")
		cleaned = strings.ReplaceAll(cleaned, "<td>", "")
		cleaned = strings.ReplaceAll(cleaned, "</td>", " ")
		cleaned = strings.ReplaceAll(cleaned, "<th>", "")
		cleaned = strings.ReplaceAll(cleaned, "</th>", " ")
		// Remove table wrapper tags
		cleaned = regexp.MustCompile(`</?table[^>]*>`).ReplaceAllString(cleaned, "")
		cleaned = regexp.MustCompile(`</?tbody[^>]*>`).ReplaceAllString(cleaned, "")
		cleaned = regexp.MustCompile(`</?thead[^>]*>`).ReplaceAllString(cleaned, "")
		return cleaned
	})

	return html
}

// renderMarkdown renders markdown/plaintext with glamour (reuses model's renderer).
func (m *Model) renderMarkdown(text string) string {
	if m.renderers.glamourRenderer == nil {
		// Fallback to raw text if no renderer
		return text
	}

	rendered, err := m.renderers.glamourRenderer.Render(text)
	if err != nil {
		// If rendering fails, return raw text
		return text
	}

	return strings.TrimSpace(rendered)
}

func normalizeRawForDisplay(raw string) string {
	if raw == "" {
		return ""
	}
	// Terminals handle LF better than CRLF for display.
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	return raw
}

func formatAttachmentSize(size int64) string {
	if size < 0 {
		return ""
	}
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
}

func stripZeroWidth(text string) string {
	if text == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		switch r {
		case 0x034F, 0x200B, 0x200C, 0x200D, 0x200E, 0x200F, 0x2060, 0xFEFF:
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func stripLeadingZeroWidth(text string) string {
	for len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		if r == utf8.RuneError && size == 1 {
			break
		}
		if runewidth.RuneWidth(r) == 0 || unicode.Is(unicode.Mn, r) ||
			unicode.Is(unicode.Mc, r) || unicode.Is(unicode.Me, r) {
			text = text[size:]
			continue
		}
		break
	}
	return text
}

func renderFixedLayout(height int, body, footer string) string {
	footerHeight := lipgloss.Height(footer)
	bodyHeight := max(0, height-footerHeight)
	bodyStyle := lipgloss.NewStyle().Height(bodyHeight).MaxHeight(bodyHeight)
	body = bodyStyle.Render(body)

	return lipgloss.JoinVertical(lipgloss.Left, body, footer)
}

func detailViewportHeight(height int) int {
	return max(1, height-1)
}

func attachmentViewportHeight(height int) int {
	return max(1, height-1)
}

func wrapTextLines(text string, width int, maxLines int) []string {
	if maxLines <= 0 {
		return nil
	}
	if width <= 0 {
		lines := make([]string, maxLines)
		return lines
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		lines := make([]string, maxLines)
		return lines
	}

	lines := make([]string, 0, maxLines)
	current := ""
	for i := range words {
		word := words[i]
		if current == "" {
			if lipgloss.Width(word) > width {
				current = truncateToWidth(word, width)
			} else {
				current = word
			}
			continue
		}

		candidate := current + " " + word
		if lipgloss.Width(candidate) <= width {
			current = candidate
			continue
		}

		lines = append(lines, current)
		if len(lines) == maxLines-1 {
			remaining := append([]string{word}, words[i+1:]...)
			line := strings.Join(remaining, " ")
			lines = append(lines, truncateToWidth(line, width))
			current = ""
			break
		}
		current = word
	}

	if current != "" && len(lines) < maxLines {
		lines = append(lines, truncateToWidth(current, width))
	}

	for len(lines) < maxLines {
		lines = append(lines, "")
	}
	return lines
}
