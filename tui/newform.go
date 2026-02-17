package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jackwu/vibesession/launcher"
)

// newForm field indices
const (
	fieldTool = iota
	fieldDir
	fieldMode
	fieldCount
)

type newForm struct {
	tool    int // 0 = claude, 1 = codex
	dirInput textinput.Model
	mode    int // 0 = normal, 1 = yolo
	focus   int // which field is focused
}

func newNewForm() newForm {
	di := textinput.New()
	di.Placeholder = "~/projects/my-app"
	di.CharLimit = 300
	// default to current working directory
	di.Focus()

	return newForm{
		tool:     0,
		dirInput: di,
		mode:     0,
		focus:    fieldTool,
	}
}

func (f *newForm) setCWD(cwd string) {
	f.dirInput.SetValue(cwd)
	f.dirInput.CursorEnd()
}

func (m Model) enterNewForm() (Model, tea.Cmd) {
	f := newNewForm()
	f.setCWD(m.cwd)
	f.dirInput.Blur() // tool is focused first, not dir
	m.newForm = &f
	m.mode = modeNew
	return m, nil
}

func (m Model) updateNewForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	f := m.newForm
	key := msg.String()

	// global keys
	switch key {
	case "esc":
		m.newForm = nil
		m.mode = modeList
		return m, nil

	case "tab", "down":
		f.blurCurrent()
		f.focus = (f.focus + 1) % fieldCount
		f.focusCurrent()
		return m, nil

	case "shift+tab", "up":
		f.blurCurrent()
		f.focus = (f.focus - 1 + fieldCount) % fieldCount
		f.focusCurrent()
		return m, nil

	case "enter":
		tool := "claude"
		if f.tool == 1 {
			tool = "codex"
		}
		dir := f.dirInput.Value()
		if dir == "" {
			dir = m.cwd
		}
		yolo := f.mode == 1
		cmd := launcher.BuildNewCommand(tool, dir, yolo)
		if cmd != "" {
			m.launchCmd = cmd
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}

	// field-specific keys
	switch f.focus {
	case fieldTool:
		switch key {
		case "left", "h":
			f.tool = 0
		case "right", "l":
			f.tool = 1
		}
	case fieldDir:
		var cmd tea.Cmd
		f.dirInput, cmd = f.dirInput.Update(msg)
		return m, cmd
	case fieldMode:
		switch key {
		case "left", "h":
			f.mode = 0
		case "right", "l":
			f.mode = 1
		}
	}

	return m, nil
}

func (f *newForm) blurCurrent() {
	if f.focus == fieldDir {
		f.dirInput.Blur()
	}
}

func (f *newForm) focusCurrent() {
	if f.focus == fieldDir {
		f.dirInput.Focus()
		f.dirInput.CursorEnd()
	}
}

func (m Model) viewNewForm() string {
	f := m.newForm

	// box styles
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(56)

	titleStr := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Render("New Session")

	// Tool field
	toolLabel := m.fieldLabel("Tool:", f.focus == fieldTool)
	toolValue := m.renderRadio([]string{"Claude Code", "Codex"}, f.tool, f.focus == fieldTool)

	// Dir field
	dirLabel := m.fieldLabel("Dir:", f.focus == fieldDir)
	dirValue := f.dirInput.View()

	// Mode field
	modeLabel := m.fieldLabel("Mode:", f.focus == fieldMode)
	modeValue := m.renderRadio([]string{"Normal", "YOLO"}, f.mode, f.focus == fieldMode)

	content := fmt.Sprintf(
		"%s\n\n%s  %s\n\n%s  %s\n\n%s  %s\n\n%s",
		titleStr,
		toolLabel, toolValue,
		dirLabel, dirValue,
		modeLabel, modeValue,
		dimStyle.Render("Enter: create  Esc: cancel  Tab: next  ←→: toggle"),
	)

	box := boxStyle.Render(content)

	// center the box on screen
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) fieldLabel(label string, focused bool) string {
	style := lipgloss.NewStyle().Width(6)
	if focused {
		style = style.Bold(true).Foreground(lipgloss.Color("39"))
	} else {
		style = style.Foreground(lipgloss.Color("252"))
	}
	return style.Render(label)
}

func (m Model) renderRadio(options []string, selected int, focused bool) string {
	var parts []string
	for i, opt := range options {
		if i == selected {
			style := lipgloss.NewStyle().Bold(true)
			if focused {
				style = style.Foreground(lipgloss.Color("39"))
			} else {
				style = style.Foreground(lipgloss.Color("255"))
			}
			parts = append(parts, style.Render("● "+opt))
		} else {
			parts = append(parts, dimStyle.Render("○ "+opt))
		}
	}
	return strings.Join(parts, "   ")
}
