package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jackwu/vibesession/launcher"
	"github.com/jackwu/vibesession/model"
)

type mode int

const (
	modeList mode = iota
	modeSearch
	modeCommand
)

type Model struct {
	sessions    []model.Session
	filtered    []model.Session
	cursor      int
	offset      int // scroll offset
	width       int
	height      int
	mode        mode
	searchInput textinput.Model
	cmdInput    textinput.Model
	filter      string // "all", "claude", "codex"
	launchCmd   string // final command to execute
	quitting    bool
}

func NewModel(sessions []model.Session) Model {
	// sort by time descending
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Time.After(sessions[j].Time)
	})

	si := textinput.New()
	si.Placeholder = "search..."
	si.CharLimit = 100

	ci := textinput.New()
	ci.CharLimit = 500

	m := Model{
		sessions:    sessions,
		filter:      "all",
		searchInput: si,
		cmdInput:    ci,
		width:       120,
		height:      30,
	}
	m.applyFilter()
	return m
}

func (m *Model) applyFilter() {
	m.filtered = nil
	search := strings.ToLower(m.searchInput.Value())

	for _, s := range m.sessions {
		// source filter
		switch m.filter {
		case "claude":
			if s.Source != model.SourceClaude {
				continue
			}
		case "codex":
			if s.Source != model.SourceCodex {
				continue
			}
		}

		// text search
		if search != "" {
			haystack := strings.ToLower(s.Summary + " " + s.Project + " " + s.ID + " " + s.TeamName)
			if !strings.Contains(haystack, search) {
				continue
			}
		}

		m.filtered = append(m.filtered, s)
	}

	// reset cursor
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	m.clampOffset()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampOffset()
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeList:
			return m.updateList(msg)
		case modeSearch:
			return m.updateSearch(msg)
		case modeCommand:
			return m.updateCommand(msg)
		}
	}
	return m, nil
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.clampOffset()
		}

	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.clampOffset()
		}

	case "home", "g":
		m.cursor = 0
		m.clampOffset()

	case "end", "G":
		m.cursor = max(0, len(m.filtered)-1)
		m.clampOffset()

	case "pgup":
		visible := m.visibleRows()
		m.cursor -= visible
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.clampOffset()

	case "pgdown":
		visible := m.visibleRows()
		m.cursor += visible
		if m.cursor >= len(m.filtered) {
			m.cursor = len(m.filtered) - 1
		}
		m.clampOffset()

	case "enter":
		if len(m.filtered) > 0 {
			s := m.filtered[m.cursor]
			cmd := launcher.BuildCommand(s)
			m.cmdInput.SetValue(cmd)
			m.cmdInput.Focus()
			m.cmdInput.CursorEnd()
			m.mode = modeCommand
		}

	case "/":
		m.searchInput.Focus()
		m.mode = modeSearch

	case "tab":
		switch m.filter {
		case "all":
			m.filter = "claude"
		case "claude":
			m.filter = "codex"
		case "codex":
			m.filter = "all"
		}
		m.applyFilter()
	}

	return m, nil
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.searchInput.Blur()
		m.mode = modeList
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.applyFilter()
	return m, cmd
}

func (m Model) updateCommand(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.cmdInput.Blur()
		m.mode = modeList
		return m, nil

	case "enter":
		m.launchCmd = m.cmdInput.Value()
		m.quitting = true
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.cmdInput, cmd = m.cmdInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// title bar
	title := titleStyle.Render("VibeSession")
	filterInfo := dimStyle.Render(fmt.Sprintf("  [%s]  %d sessions", m.filter, len(m.filtered)))
	b.WriteString(title + filterInfo + "\n")

	// header row
	header := m.renderHeader()
	b.WriteString(header + "\n")

	// session rows
	visible := m.visibleRows()
	end := m.offset + visible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := m.offset; i < end; i++ {
		s := m.filtered[i]
		row := m.renderRow(s, i == m.cursor)
		b.WriteString(row + "\n")
	}

	// pad remaining rows
	rendered := end - m.offset
	for i := rendered; i < visible; i++ {
		b.WriteString("\n")
	}

	// bottom bar
	switch m.mode {
	case modeSearch:
		b.WriteString(statusBarStyle.Render("Search: ") + m.searchInput.View())
	case modeCommand:
		b.WriteString(statusBarStyle.Render("Command: ") + m.cmdInput.View())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  Enter: execute  Esc: cancel"))
	default:
		b.WriteString(m.renderHelp())
	}

	return b.String()
}

func (m Model) renderHeader() string {
	w := m.colWidths()
	cols := []string{
		pad("Source", w.source),
		pad("Session ID", w.id),
		pad("Time", w.time),
		pad("Project", w.project),
		pad("Summary", w.summary),
	}
	return headerStyle.Render(strings.Join(cols, " "))
}

func (m Model) renderRow(s model.Session, selected bool) string {
	w := m.colWidths()

	var sourceStr string
	switch s.Source {
	case model.SourceClaude:
		sourceStr = claudeTag.Render(pad("Claude", w.source))
	case model.SourceCodex:
		sourceStr = codexTag.Render(pad("Codex", w.source))
	}

	timeStr := s.Time.Format("01-02 15:04")
	summaryStr := s.Summary
	if s.TeamName != "" {
		summaryStr = "[team:" + s.TeamName + "] " + summaryStr
	}
	summaryRunes := []rune(summaryStr)
	if len(summaryRunes) > w.summary {
		summaryStr = string(summaryRunes[:w.summary-2]) + ".."
	}

	cols := []string{
		sourceStr,
		pad(s.ShortID, w.id),
		pad(timeStr, w.time),
		pad(s.Project, w.project),
		summaryStr,
	}

	row := strings.Join(cols, " ")

	if selected {
		// re-render with selected style, stripping existing styles on source
		plainSource := pad(string(s.Source), w.source)
		plainCols := []string{
			plainSource,
			pad(s.ShortID, w.id),
			pad(timeStr, w.time),
			pad(s.Project, w.project),
			summaryStr,
		}
		row = selectedStyle.Render(strings.Join(plainCols, " "))
		// pad to full width
		row = lipgloss.PlaceHorizontal(m.width, lipgloss.Left, row)
	}

	return row
}

func (m Model) renderHelp() string {
	return helpStyle.Render("  Enter: open  /: search  Tab: filter  q: quit")
}

type colWidths struct {
	source  int
	id      int
	time    int
	project int
	summary int
}

func (m Model) colWidths() colWidths {
	w := colWidths{
		source:  7,
		id:      10,
		time:    12,
		project: 14,
	}
	// summary gets remaining width
	used := w.source + w.id + w.time + w.project + 6 // 6 for separators and padding
	w.summary = m.width - used
	if w.summary < 20 {
		w.summary = 20
	}
	return w
}

func (m Model) visibleRows() int {
	// total height minus title, header, bottom bar (3-4 lines)
	rows := m.height - 4
	if m.mode == modeCommand {
		rows -= 1 // extra line for command help
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m *Model) clampOffset() {
	visible := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// LaunchCmd returns the command to execute after TUI exits.
func (m Model) LaunchCmd() string {
	return m.launchCmd
}

func pad(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return string(runes[:width])
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
