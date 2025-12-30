package tui

import (
	"context"
	"fmt"
	"os"

	md "github.com/JohannesKaufmann/html-to-markdown"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"

	"go.withmatt.com/inbox/internal/config"
	"go.withmatt.com/inbox/internal/gmail"
)

type viewState int

const (
	viewList viewState = iota
	viewDetail
	viewImage
	viewAttachment
)

type messageViewMode int

const (
	viewModeText messageViewMode = iota
	viewModeHTML
	viewModeRaw
)

// Model is the TUI application state
type Model struct {
	currentView viewState

	ui          uiState
	inbox       inboxState
	detail      detailState
	attachments attachmentState
	image       imageState
	renderers   renderersState
	search      searchState
	theme       config.Theme
	uiConfig    config.UIConfig
	keyMapCfg   config.KeyMap

	// Gmail clients for fetching data (one per account)
	clients       []*gmail.Client
	accountNames  []string // Account names corresponding to clients
	accountBadges []AccountBadge

	// Context for cancellation
	ctx   context.Context
	debug bool
}

// New creates a new TUI model
func New(
	ctx context.Context,
	clients []*gmail.Client,
	accountNames []string,
	accountBadges []AccountBadge,
	theme config.Theme,
	uiConfig config.UIConfig,
	keyMapCfg config.KeyMap,
) Model {
	debug := os.Getenv("INBOX_DEBUG") != ""

	ui := newUIState()
	ui.help = newHelpModel(theme)
	ui.alert = newAlertModel(theme, 0)

	// Create glamour renderer once for reuse
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(markdownStyle(theme)),
		glamour.WithEmoji(),
		glamour.WithWordWrap(80), // Default, will be updated based on terminal width
	)

	// Create HTML to markdown converter with options
	converter := md.NewConverter("", true, &md.Options{
		StrongDelimiter: "**",
		EmDelimiter:     "_",
		CodeBlockStyle:  "fenced",
		LinkStyle:       "inlined",
		EscapeMode:      "disabled",
	})

	model := Model{
		currentView: viewList,
		ui:          ui,
		inbox: inboxState{
			threads:      nil, // Will be loaded async
			cursor:       0,
			scrollOffset: 0,
			loading:      true,
			selected:     make(map[string]struct{}),
		},
		detail:        newDetailState(),
		search:        newSearchState(theme),
		theme:         theme,
		uiConfig:      uiConfig,
		keyMapCfg:     keyMapCfg,
		clients:       clients,
		accountNames:  accountNames,
		accountBadges: accountBadges,
		renderers: renderersState{
			glamourRenderer: r,
			htmlConverter:   converter,
		},
		ctx:   ctx,
		debug: debug,
	}
	model.logf("debug logging enabled")
	return model
}

// Init initializes the TUI and kicks off inbox loading
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadInboxCmd(inboxLoadInit),
		m.ui.spinner.Tick,
		m.ui.alert.Init(),
		m.autoRefreshCmd(),
		m.setWindowTitleCmd(),
	)
}

// Run starts the TUI
func Run(
	ctx context.Context,
	clients []*gmail.Client,
	accountNames []string,
	accountBadges []AccountBadge,
	theme config.Theme,
	uiConfig config.UIConfig,
	keyMapCfg config.KeyMap,
) error {
	// Set up logging
	logFile := "/tmp/inbox-debug.log"
	f, err := tea.LogToFile(logFile, "inbox")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	p := tea.NewProgram(
		New(ctx, clients, accountNames, accountBadges, theme, uiConfig, keyMapCfg),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithContext(ctx),
	)
	_, err = p.Run()
	return err
}
