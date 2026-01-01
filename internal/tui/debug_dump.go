package tui

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	"github.com/adrg/xdg"
	"github.com/muesli/reflow/wordwrap"

	"go.withmatt.com/inbox/internal/log"
)

func (m *Model) debugDumpRender(name string, content string) {
	m.debugDumpFile(name+".bin", []byte(content))
}

func (m *Model) debugDumpText(name string, content string) {
	m.debugDumpFile(name+".txt", []byte(content))
}

func (m *Model) debugDumpFile(name string, content []byte) {
	if !log.DebugEnabled() {
		return
	}
	if m.ui.debugDumpHashes == nil {
		m.ui.debugDumpHashes = make(map[string][32]byte)
	}

	hash := sha256.Sum256(content)
	if prev, ok := m.ui.debugDumpHashes[name]; ok && prev == hash {
		return
	}
	m.ui.debugDumpHashes[name] = hash

	path, err := xdg.StateFile("inbox/" + name)
	if err != nil {
		m.logf("debug dump path error name=%s err=%v", name, err)
		return
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		m.logf("debug dump write error name=%s err=%v", name, err)
		return
	}
	m.logf("debug dump wrote name=%s bytes=%d path=%q", name, len(content), path)
}

func (m *Model) debugDumpCurrentMessage() {
	if !log.DebugEnabled() {
		return
	}
	if m.detail.selectedMessageIdx < 0 || m.detail.selectedMessageIdx >= len(m.detail.messages) {
		m.logf("debug dump message skipped: no selected message")
		return
	}

	msg := m.detail.messages[m.detail.selectedMessageIdx]
	m.logf("debug dump message id=%s view=%d", msg.ID, m.detail.messageViewMode)

	m.debugDumpText("message-body-text", msg.BodyText)
	m.debugDumpText("message-body-html", msg.BodyHTML)
	m.debugDumpText("message-body-raw", msg.Raw)

	if msg.BodyHTML != "" {
		cleanedHTML := cleanHTMLForConversion(msg.BodyHTML)
		markdown, err := m.renderers.htmlConverter.ConvertString(cleanedHTML)
		if err != nil {
			m.logf("debug dump markdown error id=%s err=%v", msg.ID, err)
		} else {
			m.debugDumpText("message-body-markdown", markdown)
		}
	}

	contentWidth := max(m.ui.width-2, 0)
	effectiveMode := normalizeMessageViewMode(m.detail.messageViewMode, msg)

	var bodyInput string
	var rendered string
	switch {
	case effectiveMode == viewModeRaw:
		rawText := msg.Raw
		if rawText == "" {
			if m.detail.rawLoading[msg.ID] {
				rawText = "[Loading raw message...]"
			} else {
				rawText = "[Raw message unavailable]"
			}
		}
		bodyInput = normalizeRawForDisplay(rawText)
		rendered = bodyInput
		if contentWidth > 0 {
			rendered = wordwrap.String(rendered, contentWidth)
		}
	case effectiveMode == viewModeHTML && msg.BodyHTML != "":
		cleanedHTML := cleanHTMLForConversion(msg.BodyHTML)
		markdown, err := m.renderers.htmlConverter.ConvertString(cleanedHTML)
		if err != nil {
			bodyInput = msg.BodyHTML
			rendered = msg.BodyHTML
		} else {
			bodyInput = markdown
			rendered = m.renderMarkdown(markdown, contentWidth)
		}
	case effectiveMode == viewModeText && msg.BodyText != "":
		bodyInput = msg.BodyText
		rendered = m.renderMarkdown(msg.BodyText, contentWidth)
	case msg.BodyHTML != "":
		cleanedHTML := cleanHTMLForConversion(msg.BodyHTML)
		markdown, err := m.renderers.htmlConverter.ConvertString(cleanedHTML)
		if err != nil {
			bodyInput = msg.BodyHTML
			rendered = msg.BodyHTML
		} else {
			bodyInput = markdown
			rendered = m.renderMarkdown(markdown, contentWidth)
		}
	default:
		bodyInput = "[No message body]"
		rendered = bodyInput
	}

	m.debugDumpText("message-body-input", bodyInput)
	m.debugDumpRender("message-body-rendered", rendered)
	m.debugDumpText("message-body-meta", fmt.Sprintf(
		"mode=%d effective_mode=%d width=%d has_text=%t has_html=%t has_raw=%t",
		m.detail.messageViewMode,
		effectiveMode,
		contentWidth,
		strings.TrimSpace(msg.BodyText) != "",
		strings.TrimSpace(msg.BodyHTML) != "",
		strings.TrimSpace(msg.Raw) != "",
	))
}
