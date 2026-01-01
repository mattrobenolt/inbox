package config

import (
	"fmt"
	"strings"

	"go.withmatt.com/themes"
)

type Theme struct {
	Name       string          `toml:"name"`
	Status     ThemeStatus     `toml:"status"`
	List       ThemeList       `toml:"list"`
	Detail     ThemeDetail     `toml:"detail"`
	Attachment ThemeAttachment `toml:"attachment"`
	Image      ThemeImage      `toml:"image"`
	Modal      ThemeModal      `toml:"modal"`
}

type ThemeStatus struct {
	Bg     string `toml:"bg"`
	Fg     string `toml:"fg"`
	Dim    string `toml:"dim"`
	ModeBg string `toml:"mode_bg"`
	ModeFg string `toml:"mode_fg"`
	TabBg  string `toml:"tab_bg"`
	TabFg  string `toml:"tab_fg"`
}

type ThemeList struct {
	UnreadFg   string `toml:"unread_fg"`
	SelectedFg string `toml:"selected_fg"`
	ReadFg     string `toml:"read_fg"`
	SelectedBg string `toml:"selected_bg"`
}

type ThemeDetail struct {
	SnippetFg      string `toml:"snippet_fg"`
	BorderSelected string `toml:"border_selected"`
	BorderNormal   string `toml:"border_normal"`
	HeaderLabelFg  string `toml:"header_label_fg"`
	HeaderValueFg  string `toml:"header_value_fg"`
	LinkFg         string `toml:"link_fg"`
	ViewModeBg     string `toml:"view_mode_bg"`
	ViewModeFg     string `toml:"view_mode_fg"`
}

type ThemeAttachment struct {
	TitleBg string `toml:"title_bg"`
	TitleFg string `toml:"title_fg"`
	MetaFg  string `toml:"meta_fg"`
}

type ThemeImage struct {
	ErrorFg string `toml:"error_fg"`
	MetaFg  string `toml:"meta_fg"`
}

type ThemeModal struct {
	FooterFg string `toml:"footer_fg"`
}

func ResolveTheme(theme Theme) (Theme, error) {
	palette, err := paletteForTheme(theme.Name)
	if err != nil {
		return Theme{}, err
	}
	base := themeFromPalette(palette)
	merged := mergeTheme(base, theme)
	merged = resolveThemeColorNames(merged, palette)
	merged.Name = theme.Name
	return merged, nil
}

func themeFromPalette(palette *themes.Theme) Theme {
	dim := firstNonEmpty(
		// palette.BrightWhite,
		// palette.BrightBlack,
		palette.Foreground,
	)
	modeBg := firstNonEmpty(
		palette.Magenta,
		palette.Foreground,
	)
	modeFg := firstNonEmpty(
		palette.Background,
	)
	tabBg := firstNonEmpty(
		palette.Magenta,
		palette.Foreground,
	)
	tabFg := firstNonEmpty(
		palette.Background,
	)
	unreadFg := firstNonEmpty(
		palette.BrightMagenta,
		palette.Magenta,
		palette.Foreground,
	)
	viewModeBg := firstNonEmpty(
		palette.Green,
		palette.Foreground,
	)
	viewModeFg := firstNonEmpty(
		palette.Background,
	)
	selectedBg := firstNonEmpty(
		// palette.BrightBlack,
		// palette.Black,
		palette.Background,
	)
	selectedFg := firstNonEmpty(
		palette.BrightGreen,
		palette.Green,
		palette.Foreground,
	)
	linkFg := firstNonEmpty(
		palette.Cyan,
		palette.BrightCyan,
		palette.Blue,
		palette.Foreground,
	)
	borderNormal := firstNonEmpty(
		// palette.BrightBlack,
		// palette.Black,
		palette.Background,
	)
	errorFg := firstNonEmpty(
		palette.Red,
		palette.Foreground,
	)
	return Theme{
		Status: ThemeStatus{
			Bg:     palette.Background,
			Fg:     palette.Foreground,
			Dim:    dim,
			ModeBg: modeBg,
			ModeFg: modeFg,
			TabBg:  tabBg,
			TabFg:  tabFg,
		},
		List: ThemeList{
			UnreadFg:   unreadFg,
			SelectedFg: selectedFg,
			ReadFg:     palette.Foreground,
			SelectedBg: selectedBg,
		},
		Detail: ThemeDetail{
			SnippetFg:      dim,
			BorderSelected: modeBg,
			BorderNormal:   borderNormal,
			HeaderLabelFg:  dim,
			HeaderValueFg:  palette.Foreground,
			LinkFg:         linkFg,
			ViewModeBg:     viewModeBg,
			ViewModeFg:     viewModeFg,
		},
		Attachment: ThemeAttachment{
			TitleBg: modeBg,
			TitleFg: modeFg,
			MetaFg:  dim,
		},
		Image: ThemeImage{
			ErrorFg: errorFg,
			MetaFg:  dim,
		},
		Modal: ThemeModal{
			FooterFg: dim,
		},
	}
}

func mergeTheme(base, override Theme) Theme {
	out := override
	fillIfEmpty(&out.Status.Bg, base.Status.Bg)
	fillIfEmpty(&out.Status.Fg, base.Status.Fg)
	fillIfEmpty(&out.Status.Dim, base.Status.Dim)
	fillIfEmpty(&out.Status.ModeBg, base.Status.ModeBg)
	fillIfEmpty(&out.Status.ModeFg, base.Status.ModeFg)
	fillIfEmpty(&out.Status.TabBg, base.Status.TabBg)
	fillIfEmpty(&out.Status.TabFg, base.Status.TabFg)

	fillIfEmpty(&out.List.UnreadFg, base.List.UnreadFg)
	fillIfEmpty(&out.List.SelectedFg, base.List.SelectedFg)
	fillIfEmpty(&out.List.ReadFg, base.List.ReadFg)
	fillIfEmpty(&out.List.SelectedBg, base.List.SelectedBg)

	fillIfEmpty(&out.Detail.SnippetFg, base.Detail.SnippetFg)
	fillIfEmpty(&out.Detail.BorderSelected, base.Detail.BorderSelected)
	fillIfEmpty(&out.Detail.BorderNormal, base.Detail.BorderNormal)
	fillIfEmpty(&out.Detail.HeaderLabelFg, base.Detail.HeaderLabelFg)
	fillIfEmpty(&out.Detail.HeaderValueFg, base.Detail.HeaderValueFg)
	fillIfEmpty(&out.Detail.LinkFg, base.Detail.LinkFg)
	fillIfEmpty(&out.Detail.ViewModeBg, base.Detail.ViewModeBg)
	fillIfEmpty(&out.Detail.ViewModeFg, base.Detail.ViewModeFg)

	fillIfEmpty(&out.Attachment.TitleBg, base.Attachment.TitleBg)
	fillIfEmpty(&out.Attachment.TitleFg, base.Attachment.TitleFg)
	fillIfEmpty(&out.Attachment.MetaFg, base.Attachment.MetaFg)

	fillIfEmpty(&out.Image.ErrorFg, base.Image.ErrorFg)
	fillIfEmpty(&out.Image.MetaFg, base.Image.MetaFg)

	fillIfEmpty(&out.Modal.FooterFg, base.Modal.FooterFg)

	return out
}

func fillIfEmpty(target *string, value string) {
	if *target == "" {
		*target = value
	}
}

func firstNonEmpty(candidates ...string) string {
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return ""
}

func ResolveColor(value string, theme Theme) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	palette, err := paletteForTheme(theme.Name)
	if err != nil {
		return "", err
	}
	return resolveColorName(value, palette), nil
}

func paletteForTheme(name string) (*themes.Theme, error) {
	themeName := strings.TrimSpace(name)
	if themeName == "" {
		themeName = "Nord"
	}
	palette, err := themes.GetTheme(themeName)
	if err != nil {
		return nil, fmt.Errorf("theme %q: %w", themeName, err)
	}
	return palette, nil
}

func resolveThemeColorNames(theme Theme, palette *themes.Theme) Theme {
	theme.Status.Bg = resolveColorName(theme.Status.Bg, palette)
	theme.Status.Fg = resolveColorName(theme.Status.Fg, palette)
	theme.Status.Dim = resolveColorName(theme.Status.Dim, palette)
	theme.Status.ModeBg = resolveColorName(theme.Status.ModeBg, palette)
	theme.Status.ModeFg = resolveColorName(theme.Status.ModeFg, palette)
	theme.Status.TabBg = resolveColorName(theme.Status.TabBg, palette)
	theme.Status.TabFg = resolveColorName(theme.Status.TabFg, palette)

	theme.List.UnreadFg = resolveColorName(theme.List.UnreadFg, palette)
	theme.List.SelectedFg = resolveColorName(theme.List.SelectedFg, palette)
	theme.List.ReadFg = resolveColorName(theme.List.ReadFg, palette)
	theme.List.SelectedBg = resolveColorName(theme.List.SelectedBg, palette)

	theme.Detail.SnippetFg = resolveColorName(theme.Detail.SnippetFg, palette)
	theme.Detail.BorderSelected = resolveColorName(theme.Detail.BorderSelected, palette)
	theme.Detail.BorderNormal = resolveColorName(theme.Detail.BorderNormal, palette)
	theme.Detail.HeaderLabelFg = resolveColorName(theme.Detail.HeaderLabelFg, palette)
	theme.Detail.HeaderValueFg = resolveColorName(theme.Detail.HeaderValueFg, palette)
	theme.Detail.LinkFg = resolveColorName(theme.Detail.LinkFg, palette)
	theme.Detail.ViewModeBg = resolveColorName(theme.Detail.ViewModeBg, palette)
	theme.Detail.ViewModeFg = resolveColorName(theme.Detail.ViewModeFg, palette)

	theme.Attachment.TitleBg = resolveColorName(theme.Attachment.TitleBg, palette)
	theme.Attachment.TitleFg = resolveColorName(theme.Attachment.TitleFg, palette)
	theme.Attachment.MetaFg = resolveColorName(theme.Attachment.MetaFg, palette)

	theme.Image.ErrorFg = resolveColorName(theme.Image.ErrorFg, palette)
	theme.Image.MetaFg = resolveColorName(theme.Image.MetaFg, palette)

	theme.Modal.FooterFg = resolveColorName(theme.Modal.FooterFg, palette)

	return theme
}

func resolveColorName(value string, palette *themes.Theme) string {
	if palette == nil {
		return value
	}
	key := normalizeColorName(value)
	if key == "" {
		return value
	}
	switch key {
	case "foreground":
		return palette.Foreground
	case "background":
		return palette.Background
	case "cursor":
		return palette.Cursor
	case "black":
		return palette.Black
	case "red":
		return palette.Red
	case "green":
		return palette.Green
	case "yellow":
		return palette.Yellow
	case "blue":
		return palette.Blue
	case "magenta":
		return palette.Magenta
	case "cyan":
		return palette.Cyan
	case "white":
		return palette.White
	case "brightblack":
		return palette.BrightBlack
	case "brightred":
		return palette.BrightRed
	case "brightgreen":
		return palette.BrightGreen
	case "brightyellow":
		return palette.BrightYellow
	case "brightblue":
		return palette.BrightBlue
	case "brightmagenta":
		return palette.BrightMagenta
	case "brightcyan":
		return palette.BrightCyan
	case "brightwhite":
		return palette.BrightWhite
	default:
		return value
	}
}

func normalizeColorName(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "_", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	return normalized
}
