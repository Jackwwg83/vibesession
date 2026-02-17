package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jackwu/vibesession/launcher"
	"github.com/jackwu/vibesession/model"
	"github.com/jackwu/vibesession/scanner"
)

// messagesLoadedMsg is sent when async message parsing completes.
type messagesLoadedMsg struct {
	filePath string // identifies which session this result belongs to
	messages []model.Message
}

func loadMessages(s model.Session) tea.Cmd {
	filePath := s.FilePath
	source := s.Source
	return func() tea.Msg {
		msgs := scanner.ParseMessages(filePath, source)
		return messagesLoadedMsg{filePath: filePath, messages: msgs}
	}
}

func (m Model) enterDetail() (Model, tea.Cmd) {
	if len(m.filtered) == 0 {
		return m, nil
	}
	m.detailSession = m.filtered[m.cursor]
	m.detailMessages = nil
	m.detailLines = nil
	m.detailOffset = 0
	m.detailLoading = true
	m.detailSearchInput = textinput.New()
	m.detailSearchInput.Placeholder = "search..."
	m.detailSearchInput.CharLimit = 100
	m.detailSearchQuery = ""
	m.detailMatches = nil
	m.detailMatchIdx = 0
	m.mode = modeDetail
	return m, loadMessages(m.detailSession)
}

func (m Model) updateDetailLoaded(filePath string, msgs []model.Message) Model {
	// discard stale result if user already switched to a different session
	if filePath != m.detailSession.FilePath {
		return m
	}
	m.detailMessages = msgs
	m.detailLoading = false
	m.detailLines = m.renderDetailContent()
	m.detailOffset = 0
	return m
}

func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc", "q":
		m.mode = modeList
		return m, nil

	case "enter":
		cmd := launcher.BuildCommand(m.detailSession)
		m.cmdInput.SetValue(cmd)
		m.cmdInput.Focus()
		m.cmdInput.CursorEnd()
		m.prevMode = modeDetail
		m.mode = modeCommand
		return m, nil

	case "up", "k":
		m.detailScrollUp(1)
	case "down", "j":
		m.detailScrollDown(1)
	case "pgup", "u":
		m.detailScrollUp(m.detailVisibleRows())
	case "pgdown", "d":
		m.detailScrollDown(m.detailVisibleRows())
	case "home", "g":
		m.detailOffset = 0
	case "end", "G":
		m.detailScrollToBottom()

	case "/":
		m.detailSearchInput.SetValue("")
		m.detailSearchInput.Focus()
		m.mode = modeDetailSearch
		return m, nil

	case "n":
		m.detailNextMatch()
	case "N":
		m.detailPrevMatch()
	}

	return m, nil
}

func (m Model) updateDetailSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.detailSearchInput.Blur()
		m.mode = modeDetail
		return m, nil
	case "enter":
		m.detailSearchInput.Blur()
		m.detailSearchQuery = m.detailSearchInput.Value()
		m.computeSearchMatches()
		m.mode = modeDetail
		return m, nil
	}

	var cmd tea.Cmd
	m.detailSearchInput, cmd = m.detailSearchInput.Update(msg)
	return m, cmd
}

func (m Model) viewDetail() string {
	var b strings.Builder

	// title bar
	title := detailTitleStyle.Render(fmt.Sprintf(" %s — %s — %s",
		m.detailSession.Source,
		m.detailSession.ShortID,
		m.detailSession.Project,
	))
	b.WriteString(title)
	b.WriteString("\n")

	if m.detailLoading {
		visible := m.detailVisibleRows()
		b.WriteString("\n  Loading...\n")
		for i := 2; i < visible; i++ {
			b.WriteString("\n")
		}
		b.WriteString(m.detailHelpBar())
		return b.String()
	}

	if len(m.detailLines) == 0 {
		visible := m.detailVisibleRows()
		b.WriteString("\n  No messages found.\n")
		for i := 2; i < visible; i++ {
			b.WriteString("\n")
		}
		b.WriteString(m.detailHelpBar())
		return b.String()
	}

	// render visible lines
	visible := m.detailVisibleRows()
	end := m.detailOffset + visible
	if end > len(m.detailLines) {
		end = len(m.detailLines)
	}

	for i := m.detailOffset; i < end; i++ {
		line := m.detailLines[i]
		// highlight search matches
		if m.detailSearchQuery != "" {
			line = m.highlightLine(line, i)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	// pad remaining rows
	rendered := end - m.detailOffset
	for i := rendered; i < visible; i++ {
		b.WriteString("\n")
	}

	// bottom bar
	b.WriteString(m.detailHelpBar())

	return b.String()
}

func (m Model) detailHelpBar() string {
	switch m.mode {
	case modeDetailSearch:
		return statusBarStyle.Render("Search: ") + m.detailSearchInput.View()
	default:
		info := ""
		if m.detailSearchQuery != "" && len(m.detailMatches) > 0 {
			info = dimStyle.Render(fmt.Sprintf("  Match %d/%d", m.detailMatchIdx+1, len(m.detailMatches)))
		} else if m.detailSearchQuery != "" {
			info = dimStyle.Render("  No matches")
		}
		scroll := ""
		if len(m.detailLines) > 0 {
			pct := 0
			if len(m.detailLines) > m.detailVisibleRows() {
				pct = m.detailOffset * 100 / (len(m.detailLines) - m.detailVisibleRows())
			}
			scroll = dimStyle.Render(fmt.Sprintf("  %d%%", pct))
		}
		return helpStyle.Render("  Esc: back  Enter: open  /: search  j/k: scroll") + info + scroll
	}
}

func (m Model) detailVisibleRows() int {
	// title bar + bottom bar = 2 lines
	rows := m.height - 2
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m *Model) detailScrollUp(n int) {
	m.detailOffset -= n
	if m.detailOffset < 0 {
		m.detailOffset = 0
	}
}

func (m *Model) detailScrollDown(n int) {
	m.detailOffset += n
	maxOffset := len(m.detailLines) - m.detailVisibleRows()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.detailOffset > maxOffset {
		m.detailOffset = maxOffset
	}
}

func (m *Model) detailScrollToBottom() {
	maxOffset := len(m.detailLines) - m.detailVisibleRows()
	if maxOffset < 0 {
		maxOffset = 0
	}
	m.detailOffset = maxOffset
}

// renderDetailContent renders all messages into lines for the viewport.
func (m Model) renderDetailContent() []string {
	var lines []string
	maxWidth := m.width - 2 // small margin
	if maxWidth < 40 {
		maxWidth = 40
	}

	for _, msg := range m.detailMessages {
		// role header
		var header string
		switch msg.Role {
		case "user":
			header = userRoleStyle.Render(pad(" USER", maxWidth))
		case "assistant":
			header = assistantRoleStyle.Render(pad(" ASSISTANT", maxWidth))
		}
		lines = append(lines, header)

		// message text
		if msg.Text != "" {
			textStyle := lipgloss.NewStyle()
			if msg.Role == "assistant" {
				textStyle = textStyle.Foreground(lipgloss.Color("250"))
			}
			wrapped := wrapText(msg.Text, maxWidth-2)
			for _, wl := range wrapped {
				lines = append(lines, " "+textStyle.Render(wl))
			}
		}

		// tool calls
		for _, tc := range msg.ToolCalls {
			lines = append(lines, " "+toolCallStyle.Render("[Tool: "+tc+"]"))
		}

		// blank separator
		lines = append(lines, "")
	}

	return lines
}

// wrapText splits text into lines that fit within maxWidth.
func wrapText(text string, maxWidth int) []string {
	var result []string
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			result = append(result, "")
			continue
		}
		runes := []rune(line)
		for len(runes) > maxWidth {
			result = append(result, string(runes[:maxWidth]))
			runes = runes[maxWidth:]
		}
		result = append(result, string(runes))
	}
	return result
}

// Search match support

func (m *Model) computeSearchMatches() {
	m.detailMatches = nil
	m.detailMatchIdx = 0
	query := strings.ToLower(m.detailSearchQuery)
	if query == "" {
		return
	}
	for i, line := range m.detailLines {
		if strings.Contains(strings.ToLower(line), query) {
			m.detailMatches = append(m.detailMatches, i)
		}
	}
	// jump to first match
	if len(m.detailMatches) > 0 {
		m.detailScrollToMatch(0)
	}
}

func (m *Model) detailNextMatch() {
	if len(m.detailMatches) == 0 {
		return
	}
	m.detailMatchIdx = (m.detailMatchIdx + 1) % len(m.detailMatches)
	m.detailScrollToMatch(m.detailMatchIdx)
}

func (m *Model) detailPrevMatch() {
	if len(m.detailMatches) == 0 {
		return
	}
	m.detailMatchIdx--
	if m.detailMatchIdx < 0 {
		m.detailMatchIdx = len(m.detailMatches) - 1
	}
	m.detailScrollToMatch(m.detailMatchIdx)
}

func (m *Model) detailScrollToMatch(idx int) {
	lineNum := m.detailMatches[idx]
	// center the match in the viewport
	visible := m.detailVisibleRows()
	m.detailOffset = lineNum - visible/2
	if m.detailOffset < 0 {
		m.detailOffset = 0
	}
	maxOffset := len(m.detailLines) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.detailOffset > maxOffset {
		m.detailOffset = maxOffset
	}
}

func (m Model) highlightLine(line string, lineIdx int) string {
	// check if this line is the current match
	isCurrentMatch := false
	for i, matchLine := range m.detailMatches {
		if matchLine == lineIdx && i == m.detailMatchIdx {
			isCurrentMatch = true
			break
		}
	}
	if isCurrentMatch {
		return searchHighlightStyle.Render(line)
	}
	return line
}
