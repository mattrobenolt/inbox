package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"

	"go.withmatt.com/inbox/internal/config"
)

type listKeyMap struct {
	Up             key.Binding
	Down           key.Binding
	PageUp         key.Binding
	PageDown       key.Binding
	Open           key.Binding
	ToggleRead     key.Binding
	ToggleSelect   key.Binding
	ClearSelection key.Binding
	Archive        key.Binding
	Delete         key.Binding
	DeleteForever  key.Binding
	Undo           key.Binding
	Search         key.Binding
	Refresh        key.Binding
	Help           key.Binding
	Quit           key.Binding
}

type detailKeyMap struct {
	Up           key.Binding
	Down         key.Binding
	ToggleExpand key.Binding
	ToggleView   key.Binding
	Attachments  key.Binding
	Back         key.Binding
	Help         key.Binding
	Quit         key.Binding
}

type searchKeyMap struct {
	Submit key.Binding
	Cancel key.Binding
	Quit   key.Binding
}

type attachmentKeyMap struct {
	Back key.Binding
	Quit key.Binding
}

type imageKeyMap struct {
	Back key.Binding
	Quit key.Binding
}

type attachmentsModalKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Download key.Binding
	View     key.Binding
	Close    key.Binding
}

type keyMap struct {
	view                   viewState
	searchActive           bool
	attachmentsModalActive bool

	list                 listKeyMap
	detail               detailKeyMap
	search               searchKeyMap
	attachment           attachmentKeyMap
	image                imageKeyMap
	attachmentsModalKeys attachmentsModalKeyMap
}

func keyMapFromConfig(cfg config.KeyMap) keyMap {
	return keyMap{
		list: listKeyMap{
			Up: makeBinding(bindingDef{keys: []string{"k", "up"}, desc: "up"}, cfg.List.Up),
			Down: makeBinding(
				bindingDef{keys: []string{"j", "down"}, desc: "down"},
				cfg.List.Down,
			),
			PageUp: makeBinding(
				bindingDef{keys: []string{"pgup"}, desc: "page up"},
				cfg.List.PageUp,
			),
			PageDown: makeBinding(
				bindingDef{keys: []string{"pgdown"}, desc: "page down"},
				cfg.List.PageDown,
			),
			Open: makeBinding(
				bindingDef{keys: []string{"enter"}, desc: "open"},
				cfg.List.Open,
			),
			ToggleRead: makeBinding(
				bindingDef{keys: []string{" ", "space"}, desc: "read/unread"},
				cfg.List.ToggleRead,
			),
			ToggleSelect: makeBinding(
				bindingDef{keys: []string{"x"}, desc: "select"},
				cfg.List.ToggleSelect,
			),
			ClearSelection: makeBinding(
				bindingDef{keys: []string{"X"}, desc: "clear"},
				cfg.List.ClearSelection,
			),
			Archive: makeBinding(
				bindingDef{keys: []string{"a"}, desc: "archive"},
				cfg.List.Archive,
			),
			Delete: makeBinding(
				bindingDef{keys: []string{"d"}, desc: "trash"},
				cfg.List.Delete,
			),
			DeleteForever: makeBinding(
				bindingDef{keys: []string{"D"}, desc: "delete"},
				cfg.List.DeleteForever,
			),
			Undo: makeBinding(
				bindingDef{keys: []string{"u"}, desc: "undo"},
				cfg.List.Undo,
			),
			Search: makeBinding(
				bindingDef{keys: []string{"/"}, desc: "search"},
				cfg.List.Search,
			),
			Refresh: makeBinding(
				bindingDef{keys: []string{"r"}, desc: "refresh"},
				cfg.List.Refresh,
			),
			Help: makeBinding(bindingDef{keys: []string{"?"}, desc: "help"}, cfg.List.Help),
			Quit: makeBinding(
				bindingDef{keys: []string{"q", "esc", "ctrl+c"}, desc: "quit"},
				cfg.List.Quit,
			),
		},
		detail: detailKeyMap{
			Up: makeBinding(
				bindingDef{keys: []string{"k", "up"}, desc: "prev"},
				cfg.Detail.Up,
			),
			Down: makeBinding(
				bindingDef{keys: []string{"j", "down"}, desc: "next"},
				cfg.Detail.Down,
			),
			ToggleExpand: makeBinding(
				bindingDef{keys: []string{"enter", " ", "space"}, desc: "expand"},
				cfg.Detail.ToggleExpand,
			),
			ToggleView: makeBinding(
				bindingDef{keys: []string{"t"}, desc: "toggle view"},
				cfg.Detail.ToggleView,
			),
			Attachments: makeBinding(
				bindingDef{keys: []string{"a"}, desc: "attachments"},
				cfg.Detail.Attachments,
			),
			Back: makeBinding(
				bindingDef{keys: []string{"esc", "q"}, desc: "back"},
				cfg.Detail.Back,
			),
			Help: makeBinding(
				bindingDef{keys: []string{"?"}, desc: "help"},
				cfg.Detail.Help,
			),
			Quit: makeBinding(
				bindingDef{keys: []string{"ctrl+c"}, desc: "quit"},
				cfg.Detail.Quit,
			),
		},
		search: searchKeyMap{
			Submit: makeBinding(
				bindingDef{keys: []string{"enter"}, desc: "apply"},
				cfg.Search.Submit,
			),
			Cancel: makeBinding(
				bindingDef{keys: []string{"esc"}, desc: "cancel"},
				cfg.Search.Cancel,
			),
			Quit: makeBinding(
				bindingDef{keys: []string{"ctrl+c"}, desc: "quit"},
				cfg.Search.Quit,
			),
		},
		attachment: attachmentKeyMap{
			Back: makeBinding(
				bindingDef{keys: []string{"esc", "q"}, desc: "back"},
				cfg.Attachment.Back,
			),
			Quit: makeBinding(
				bindingDef{keys: []string{"ctrl+c"}, desc: "quit"},
				cfg.Attachment.Quit,
			),
		},
		image: imageKeyMap{
			Back: makeBinding(bindingDef{keys: []string{"esc", "q"}, desc: "back"}, cfg.Image.Back),
			Quit: makeBinding(bindingDef{keys: []string{"ctrl+c"}, desc: "quit"}, cfg.Image.Quit),
		},
		attachmentsModalKeys: attachmentsModalKeyMap{
			Up: makeBinding(
				bindingDef{keys: []string{"k", "up"}, desc: "up"},
				cfg.AttachmentsModal.Up,
			),
			Down: makeBinding(
				bindingDef{keys: []string{"j", "down"}, desc: "down"},
				cfg.AttachmentsModal.Down,
			),
			Download: makeBinding(
				bindingDef{keys: []string{"enter", "d"}, desc: "download"},
				cfg.AttachmentsModal.Download,
			),
			View: makeBinding(
				bindingDef{keys: []string{"v"}, desc: "view"},
				cfg.AttachmentsModal.View,
			),
			Close: makeBinding(
				bindingDef{keys: []string{"esc", "a", "q"}, desc: "close"},
				cfg.AttachmentsModal.Close,
			),
		},
	}
}

func (m Model) keyMap() keyMap {
	km := keyMapFromConfig(m.keyMapCfg)
	km.view = m.currentView
	km.searchActive = m.search.active
	km.attachmentsModalActive = m.attachments.modal.show
	return km
}

type bindingDef struct {
	keys []string
	desc string
}

func makeBinding(def bindingDef, override []string) key.Binding {
	keys := def.keys
	if len(override) > 0 {
		keys = override
	}
	return key.NewBinding(
		key.WithKeys(keys...),
		key.WithHelp(formatHelpKeys(keys), def.desc),
	)
}

func formatHelpKeys(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		label := formatKeyLabel(key)
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		out = append(out, label)
	}
	return strings.Join(out, "/")
}

func formatKeyLabel(key string) string {
	switch key {
	case "up":
		return "↑"
	case "down":
		return "↓"
	case "left":
		return "←"
	case "right":
		return "→"
	case "pgdown":
		return "pgdn"
	case "pgup":
		return "pgup"
	case " ":
		return "space"
	default:
		return key
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	if k.attachmentsModalActive {
		return []key.Binding{
			k.attachmentsModalKeys.Up,
			k.attachmentsModalKeys.Down,
			k.attachmentsModalKeys.Download,
			k.attachmentsModalKeys.View,
			k.attachmentsModalKeys.Close,
		}
	}
	if k.searchActive {
		return []key.Binding{k.search.Submit, k.search.Cancel, k.search.Quit}
	}
	switch k.view {
	case viewDetail:
		return []key.Binding{
			k.detail.Up,
			k.detail.Down,
			k.detail.ToggleExpand,
			k.detail.Attachments,
			k.detail.Back,
			k.detail.Help,
		}
	case viewAttachment:
		return []key.Binding{k.attachment.Back, k.attachment.Quit}
	case viewImage:
		return []key.Binding{k.image.Back, k.image.Quit}
	case viewList:
		return []key.Binding{
			k.list.Up,
			k.list.Down,
			k.list.Open,
			k.list.ToggleSelect,
			k.list.Archive,
			k.list.Delete,
			k.list.Undo,
			k.list.Search,
			k.list.Help,
			k.list.Quit,
		}
	default:
		return []key.Binding{
			k.list.Up,
			k.list.Down,
			k.list.Open,
			k.list.Search,
			k.list.Help,
			k.list.Quit,
		}
	}
}

func (k keyMap) FullHelp() [][]key.Binding {
	if k.attachmentsModalActive {
		return [][]key.Binding{
			{k.attachmentsModalKeys.Up, k.attachmentsModalKeys.Down},
			{k.attachmentsModalKeys.Download, k.attachmentsModalKeys.View},
			{k.attachmentsModalKeys.Close},
		}
	}
	if k.searchActive {
		return [][]key.Binding{
			{k.search.Submit, k.search.Cancel},
			{k.search.Quit},
		}
	}
	switch k.view {
	case viewDetail:
		return [][]key.Binding{
			{k.detail.Up, k.detail.Down},
			{k.detail.ToggleExpand, k.detail.ToggleView, k.detail.Attachments},
			{k.detail.Back, k.detail.Help, k.detail.Quit},
		}
	case viewAttachment:
		return [][]key.Binding{
			{k.attachment.Back, k.attachment.Quit},
		}
	case viewImage:
		return [][]key.Binding{
			{k.image.Back, k.image.Quit},
		}
	case viewList:
		return [][]key.Binding{
			{k.list.Up, k.list.Down, k.list.PageUp, k.list.PageDown},
			{k.list.Open, k.list.ToggleRead, k.list.ToggleSelect, k.list.ClearSelection},
			{k.list.Archive, k.list.Delete, k.list.DeleteForever, k.list.Undo},
			{k.list.Search, k.list.Refresh},
			{k.list.Help, k.list.Quit},
		}
	default:
		return [][]key.Binding{
			{k.list.Up, k.list.Down, k.list.PageUp, k.list.PageDown},
			{k.list.Open, k.list.ToggleRead, k.list.ToggleSelect, k.list.ClearSelection},
			{k.list.Archive, k.list.Delete, k.list.DeleteForever, k.list.Undo},
			{k.list.Search, k.list.Refresh},
			{k.list.Help, k.list.Quit},
		}
	}
}
