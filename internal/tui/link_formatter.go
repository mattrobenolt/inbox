package tui

import (
	"os"
	"strings"
	"unicode"

	"github.com/charmbracelet/glamour/ansi"
)

const (
	linkSpaceSentinel     = '\ue000'
	linkCommaSentinel     = '\ue001'
	linkPeriodSentinel    = '\ue002'
	linkSemicolonSentinel = '\ue003'
	linkDashSentinel      = '\ue004'
	linkPlusSentinel      = '\ue005'
	linkPipeSentinel      = '\ue006'
)

func smartLinkFormatter() ansi.LinkFormatter {
	return ansi.LinkFormatterFunc(func(data ansi.LinkData, ctx ansi.RenderContext) (string, error) {
		data.URL = sanitizeLinkURL(data.URL)
		if supportsOSC8() {
			data.Text = linkTextSentinelize(data.Text)
			return ansi.HyperlinkFormatter.FormatLink(data, ctx)
		}
		return ansi.DefaultFormatter.FormatLink(data, ctx)
	})
}

func sanitizeLinkURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	return strings.Join(strings.Fields(rawURL), "")
}

func linkTextSentinelize(text string) string {
	if text == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if unicode.IsSpace(r) {
			b.WriteRune(linkSpaceSentinel)
			continue
		}
		switch r {
		case ',':
			b.WriteRune(linkCommaSentinel)
		case '.':
			b.WriteRune(linkPeriodSentinel)
		case ';':
			b.WriteRune(linkSemicolonSentinel)
		case '-':
			b.WriteRune(linkDashSentinel)
		case '+':
			b.WriteRune(linkPlusSentinel)
		case '|':
			b.WriteRune(linkPipeSentinel)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func restoreLinkTextSentinels(text string) string {
	if text == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		string(linkSpaceSentinel), " ",
		string(linkCommaSentinel), ",",
		string(linkPeriodSentinel), ".",
		string(linkSemicolonSentinel), ";",
		string(linkDashSentinel), "-",
		string(linkPlusSentinel), "+",
		string(linkPipeSentinel), "|",
	)
	return replacer.Replace(text)
}

func supportsOSC8() bool {
	termProgram := os.Getenv("TERM_PROGRAM")
	if termProgram != "" {
		switch termProgram {
		case "iTerm.app", "vscode", "Windows Terminal", "WezTerm", "Hyper", "ghostty":
			return true
		}
	}

	term := os.Getenv("TERM")
	if term != "" {
		supportedTerms := []string{
			"xterm-256color",
			"screen-256color",
			"tmux-256color",
			"alacritty",
			"xterm-kitty",
			"xterm-ghostty",
		}
		for _, supportedTerm := range supportedTerms {
			if strings.Contains(term, supportedTerm) {
				return true
			}
		}
	}

	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}

	if os.Getenv("ALACRITTY_LOG") != "" || os.Getenv("ALACRITTY_SOCKET") != "" {
		return true
	}

	return false
}
