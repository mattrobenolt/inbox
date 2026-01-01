package tui

import (
	md "github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"go.dalton.dog/bubbleup"

	"go.withmatt.com/inbox/internal/config"
	"go.withmatt.com/inbox/internal/gmail"
)

type uiState struct {
	width     int
	height    int
	spinner   spinner.Model
	help      help.Model
	alert     bubbleup.AlertModel
	showHelp  bool
	showError bool
	err       error

	debugDumpHashes map[string][32]byte
}

type inboxState struct {
	threads        []gmail.Thread
	cursor         int
	scrollOffset   int
	nextPageToken  string
	loadingMore    bool
	loading        bool
	refreshing     bool
	loadingThreads int
	loadedThreads  int
	filteredIdx    []int
	selected       map[string]struct{}
	delete         deleteState
	undo           undoState
}

type detailState struct {
	currentThread        *gmail.Thread
	messages             []gmail.Message
	loading              bool
	viewport             viewport.Model
	expandedMessages     map[string]bool
	selectedMessageIdx   int
	messageViewMode      messageViewMode
	savedViewportYOffset int
	rawLoading           map[string]bool
}

type attachmentsModalState struct {
	show           bool
	selectedIdx    int
	attachments    []gmail.Attachment
	messageID      string
	accountIndex   int
	downloading    bool
	loadingPreview bool
}

type attachmentPreviewState struct {
	filename string
	mimeType string
	raw      string
	rendered string
	size     int64
}

type attachmentState struct {
	modal   attachmentsModalState
	preview attachmentPreviewState
}

type imageState struct {
	data       string
	mimeType   string
	filename   string
	size       int64
	needsClear bool
}

type renderersState struct {
	glamourRenderer *glamour.TermRenderer
	glamourWidth    int
	loggedHyperlink bool
	htmlConverter   *md.Converter
}

type searchState struct {
	active bool
	query  string
	input  textinput.Model

	previousQuery    string
	remoteLoading    bool
	remoteGeneration int
	remoteKeys       map[string]struct{}
}

type threadRef struct {
	threadID     string
	accountIndex int
}

type deleteState struct {
	pending    bool
	inProgress bool
	action     deleteAction
	targets    []threadRef
}

type deleteAction int

const (
	deleteActionTrash deleteAction = iota
	deleteActionArchive
	deleteActionPermanent
)

type undoState struct {
	action     deleteAction
	inProgress bool
	threads    []gmail.Thread
}

func newUIState() uiState {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = s.Style.Foreground(s.Style.GetForeground())
	return uiState{spinner: s}
}

func newDetailState() detailState {
	return detailState{
		viewport:         viewport.New(0, 0),
		expandedMessages: make(map[string]bool),
		messageViewMode:  viewModeHTML,
		rawLoading:       make(map[string]bool),
	}
}

func newSearchState(theme config.Theme) searchState {
	input := textinput.New()
	input.Prompt = "/ "
	input.ShowSuggestions = true
	input.SetSuggestions([]string{
		"has:attachment",
		"is:unread",
		"is:muted",
		"is:important",
		"is:starred",
		"is:read",
		"subject:",
		"from:",
		"in:anywhere",
		"in:archive",
		"in:snoozed",
		"filename:",
	})
	input.CharLimit = 200
	input.Blur()
	statusStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(theme.Status.Bg)).
		Foreground(lipgloss.Color(theme.Status.Fg)).
		Bold(true)
	input.PromptStyle = statusStyle
	input.TextStyle = statusStyle
	input.PlaceholderStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(theme.Status.Bg)).
		Foreground(lipgloss.Color(theme.Status.Dim)).
		Faint(true)
	input.Cursor.Style = lipgloss.NewStyle().
		Background(lipgloss.Color(theme.Status.Fg)).
		Foreground(lipgloss.Color(theme.Status.Bg))
	return searchState{input: input}
}

func (m *Model) resetDetail() {
	m.detail.currentThread = nil
	m.detail.messages = nil
	m.detail.loading = false
	m.detail.expandedMessages = make(map[string]bool)
	m.detail.selectedMessageIdx = 0
	m.detail.messageViewMode = viewModeHTML
	m.detail.savedViewportYOffset = 0
	m.detail.rawLoading = make(map[string]bool)
}

func (m *Model) resetAttachmentPreview() {
	m.attachments.preview = attachmentPreviewState{}
}
