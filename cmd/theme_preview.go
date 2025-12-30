package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"go.withmatt.com/themes"

	"go.withmatt.com/inbox/internal/config"
)

var previewThemeName string

var themePreviewCmd = &cobra.Command{
	Use:   "theme-preview",
	Short: "Preview resolved theme colors",
	RunE:  runThemePreview,
}

func init() {
	themePreviewCmd.Flags().StringVar(&previewThemeName, "name", "", "theme name to preview")
	rootCmd.AddCommand(themePreviewCmd)
}

func runThemePreview(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	theme := cfg.Theme
	if previewThemeName != "" {
		theme.Name = previewThemeName
	}

	resolved, err := config.ResolveTheme(theme)
	if err != nil {
		return fmt.Errorf("unable to resolve theme: %w", err)
	}

	themeLabel := resolved.Name
	if themeLabel == "" {
		themeLabel = "custom"
	}
	fmt.Printf("Theme preview: %s\n", themeLabel)

	paletteName := previewThemeName
	if paletteName == "" {
		paletteName = cfg.Theme.Name
	}
	if paletteName == "" {
		paletteName = "Nord"
	}
	palette, err := themes.GetTheme(paletteName)
	if err != nil {
		return fmt.Errorf("unable to load palette %q: %w", paletteName, err)
	}

	printThemeSection("Palette (base)", []themeColor{
		{label: "background", value: palette.Background},
		{label: "foreground", value: palette.Foreground},
		{label: "cursor", value: palette.Cursor},
	})

	printThemeSection("Palette (ansi)", []themeColor{
		{label: "black", value: palette.Black},
		{label: "red", value: palette.Red},
		{label: "green", value: palette.Green},
		{label: "yellow", value: palette.Yellow},
		{label: "blue", value: palette.Blue},
		{label: "magenta", value: palette.Magenta},
		{label: "cyan", value: palette.Cyan},
		{label: "white", value: palette.White},
	})

	printThemeSection("Palette (bright)", []themeColor{
		{label: "bright_black", value: palette.BrightBlack},
		{label: "bright_red", value: palette.BrightRed},
		{label: "bright_green", value: palette.BrightGreen},
		{label: "bright_yellow", value: palette.BrightYellow},
		{label: "bright_blue", value: palette.BrightBlue},
		{label: "bright_magenta", value: palette.BrightMagenta},
		{label: "bright_cyan", value: palette.BrightCyan},
		{label: "bright_white", value: palette.BrightWhite},
	})

	printThemeSection("Status", []themeColor{
		{label: "bg", value: resolved.Status.Bg},
		{label: "fg", value: resolved.Status.Fg},
		{label: "dim", value: resolved.Status.Dim},
		{label: "mode_bg", value: resolved.Status.ModeBg},
		{label: "mode_fg", value: resolved.Status.ModeFg},
		{label: "tab_bg", value: resolved.Status.TabBg},
		{label: "tab_fg", value: resolved.Status.TabFg},
	})

	printThemeSection("List", []themeColor{
		{label: "unread_fg", value: resolved.List.UnreadFg},
		{label: "selected_fg", value: resolved.List.SelectedFg},
		{label: "read_fg", value: resolved.List.ReadFg},
		{label: "selected_bg", value: resolved.List.SelectedBg},
	})

	printThemeSection("Detail", []themeColor{
		{label: "snippet_fg", value: resolved.Detail.SnippetFg},
		{label: "border_selected", value: resolved.Detail.BorderSelected},
		{label: "border_normal", value: resolved.Detail.BorderNormal},
		{label: "header_label_fg", value: resolved.Detail.HeaderLabelFg},
		{label: "header_value_fg", value: resolved.Detail.HeaderValueFg},
		{label: "view_mode_bg", value: resolved.Detail.ViewModeBg},
		{label: "view_mode_fg", value: resolved.Detail.ViewModeFg},
	})

	printThemeSection("Attachment", []themeColor{
		{label: "title_bg", value: resolved.Attachment.TitleBg},
		{label: "title_fg", value: resolved.Attachment.TitleFg},
		{label: "meta_fg", value: resolved.Attachment.MetaFg},
	})

	printThemeSection("Image", []themeColor{
		{label: "error_fg", value: resolved.Image.ErrorFg},
		{label: "meta_fg", value: resolved.Image.MetaFg},
	})

	printThemeSection("Modal", []themeColor{
		{label: "footer_fg", value: resolved.Modal.FooterFg},
	})

	return nil
}

type themeColor struct {
	label string
	value string
}

func printThemeSection(title string, colors []themeColor) {
	fmt.Printf("\n%s\n", title)
	for _, item := range colors {
		fmt.Printf("  %-16s %s %s\n", item.label, renderSwatch(item.value), item.value)
	}
}

func renderSwatch(color string) string {
	if color == "" {
		return "  "
	}
	return lipgloss.NewStyle().
		Background(lipgloss.Color(color)).
		Render("  ")
}
