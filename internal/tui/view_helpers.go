package tui

import (
	"fmt"
	"os"
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
func (m *Model) renderMarkdown(text string, width int) string {
	text = sanitizeImageMarkdown(text)
	text = m.linkResolver.ResolveText(m.ctx, text)
	m.ensureGlamourRenderer(width)
	if m.renderers.glamourRenderer == nil {
		return text
	}
	rendered, err := m.renderers.glamourRenderer.Render(text)
	if err != nil {
		// If rendering fails, return raw text
		return text
	}
	rendered = normalizeOSC8LineBreaks(rendered)
	rendered = restoreLinkTextSentinels(rendered)
	if !m.renderers.loggedHyperlink && strings.Contains(text, "http") {
		m.renderers.loggedHyperlink = true
		m.logf(
			"glamour osc8=%t width=%d term=%q term_program=%q",
			strings.Contains(rendered, "\x1b]8;;"),
			width,
			os.Getenv("TERM"),
			os.Getenv("TERM_PROGRAM"),
		)
	}

	return strings.TrimSpace(rendered)
}

func (m *Model) ensureGlamourRenderer(width int) {
	if width <= 0 {
		width = 80
	}
	if m.renderers.glamourRenderer != nil && m.renderers.glamourWidth == width {
		return
	}
	renderer, err := newGlamourRenderer(m.theme, width)
	if err != nil {
		m.logf("glamour renderer error: %v", err)
		return
	}
	m.renderers.glamourRenderer = renderer
	m.renderers.glamourWidth = width
}

func normalizeOSC8LineBreaks(text string) string {
	const (
		osc8Start = "\x1b]8;;"
		osc8End   = "\x1b]8;;\x1b\\"
		oscST     = "\x1b\\"
		oscBEL    = '\x07'
	)

	if !strings.Contains(text, osc8Start) {
		return text
	}

	var b strings.Builder
	b.Grow(len(text))

	currentURL := ""
	for i := 0; i < len(text); {
		if strings.HasPrefix(text[i:], osc8Start) {
			b.WriteString(osc8Start)
			i += len(osc8Start)
			urlStart := i
			for i < len(text) {
				if text[i] == oscBEL {
					rawURL := text[urlStart:i]
					url := sanitizeLinkURL(rawURL)
					b.WriteString(url)
					b.WriteByte(oscBEL)
					if url == "" {
						currentURL = ""
					} else {
						currentURL = url
					}
					i++
					break
				}
				if strings.HasPrefix(text[i:], oscST) {
					rawURL := text[urlStart:i]
					url := sanitizeLinkURL(rawURL)
					b.WriteString(url)
					b.WriteString(oscST)
					if url == "" {
						currentURL = ""
					} else {
						currentURL = url
					}
					i += len(oscST)
					break
				}
				i++
			}
			continue
		}

		if text[i] == '\n' && currentURL != "" {
			b.WriteString(osc8End)
			b.WriteByte('\n')
			b.WriteString(osc8Start)
			b.WriteString(currentURL)
			b.WriteString(oscST)
			i++
			continue
		}

		b.WriteByte(text[i])
		i++
	}

	return b.String()
}

var imageMarkdownRe = regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`)

func sanitizeImageMarkdown(markdown string) string {
	if !strings.Contains(markdown, "![") {
		return markdown
	}
	return imageMarkdownRe.ReplaceAllStringFunc(markdown, func(match string) string {
		open := strings.Index(match, "(")
		close := strings.LastIndex(match, ")")
		if open == -1 || close == -1 || close <= open+1 {
			return match
		}
		alt := ""
		if open > 2 && match[open-1] == ']' {
			alt = match[2 : open-1]
		}
		inner := strings.TrimSpace(match[open+1 : close])
		if inner == "" {
			return match
		}
		urlPart, titlePart := splitImageURLTitle(inner)
		urlPart = sanitizeLinkURL(urlPart)
		if urlPart == "" {
			return match
		}
		label := sanitizeImageAlt(alt)
		link := "[" + label + "](" + urlPart
		if titlePart != "" {
			link += " " + titlePart
		}
		link += ")"
		return link
	})
}

func splitImageURLTitle(inner string) (string, string) {
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return "", ""
	}
	if strings.HasPrefix(inner, "<") {
		if end := strings.Index(inner, ">"); end != -1 {
			urlPart := inner[1:end]
			titlePart := strings.TrimSpace(inner[end+1:])
			return urlPart, titlePart
		}
	}
	for i, r := range inner {
		if unicode.IsSpace(r) {
			urlPart := inner[:i]
			titlePart := strings.TrimSpace(inner[i:])
			return urlPart, titlePart
		}
	}
	return inner, ""
}

func sanitizeImageAlt(alt string) string {
	alt = strings.TrimSpace(alt)
	if alt == "" {
		return "Image"
	}
	alt = strings.Join(strings.Fields(alt), " ")
	return escapeMarkdownLabel(alt)
}

func escapeMarkdownLabel(label string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"[", "\\[",
		"]", "\\]",
	)
	return replacer.Replace(label)
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
