package tui

import (
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"

	"go.withmatt.com/inbox/internal/config"
)

func markdownStyle(theme config.Theme) ansi.StyleConfig {
	style := styles.DarkStyleConfig

	style.Document.Color = ptr(theme.Status.Fg)
	style.Paragraph.Color = ptr(theme.Status.Fg)
	style.Text.Color = ptr(theme.Status.Fg)

	style.BlockQuote.Color = ptr(theme.Status.Dim)
	style.BlockQuote.IndentToken = ptr("▍ ")

	style.Heading.Color = ptr(theme.Status.ModeBg)
	style.H1.Color = ptr(theme.Status.ModeBg)
	style.H1.BackgroundColor = nil
	style.H2.Color = ptr(theme.Status.ModeBg)
	style.H3.Color = ptr(theme.Status.ModeBg)
	style.H4.Color = ptr(theme.Status.ModeBg)
	style.H5.Color = ptr(theme.Status.ModeBg)
	style.H6.Color = ptr(theme.Status.ModeBg)

	style.HorizontalRule.Color = ptr(theme.Status.Dim)

	style.Link.Color = ptr(theme.Status.TabBg)
	style.LinkText.Color = ptr(theme.Status.TabBg)
	style.LinkText.Bold = ptr(true)

	style.Code.Color = ptr(theme.Status.Fg)
	style.Code.BackgroundColor = ptr(theme.Detail.BorderNormal)
	style.CodeBlock.Color = ptr(theme.Status.Fg)
	style.CodeBlock.BackgroundColor = ptr(theme.Detail.BorderNormal)

	return style
}

func ptr[T any](value T) *T {
	return &value
}
