package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackwu/vibesession/model"
	"github.com/jackwu/vibesession/scanner"
	"github.com/jackwu/vibesession/tui"
)

func main() {
	// scan both sources concurrently
	claudeCh := make(chan []model.Session)
	codexCh := make(chan []model.Session)

	go func() { claudeCh <- scanner.ScanClaude() }()
	go func() { codexCh <- scanner.ScanCodex() }()

	claudeSessions := <-claudeCh
	codexSessions := <-codexCh

	var all []model.Session
	all = append(all, claudeSessions...)
	all = append(all, codexSessions...)

	if len(all) == 0 {
		fmt.Println("No sessions found.")
		os.Exit(0)
	}

	// --list flag: print sessions as plain text (for testing / scripting)
	if len(os.Args) > 1 && os.Args[1] == "--list" {
		sort.Slice(all, func(i, j int) bool {
			return all[i].Time.After(all[j].Time)
		})
		for _, s := range all {
			summary := s.Summary
			if s.TeamName != "" {
				summary = "[team:" + s.TeamName + "] " + summary
			}
			fmt.Printf("%-6s │ %s │ %s │ %-14s │ %s\n",
				s.Source, s.ShortID, s.Time.Format("01-02 15:04"), s.Project, summary)
		}
		return
	}

	m := tui.NewModel(all)
	if cwd, err := os.Getwd(); err == nil {
		m.SetCWD(cwd)
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// after TUI exits, check if we need to launch a command
	finalModel := result.(tui.Model)
	cmd := finalModel.LaunchCmd()
	if cmd == "" {
		return
	}

	// execute the command via shell, replacing current process
	shell := "/bin/bash"
	if runtime.GOOS == "darwin" {
		if zsh, err := exec.LookPath("zsh"); err == nil {
			shell = zsh
		}
	}

	// use exec to replace current process
	execCmd := exec.Command(shell, "-c", cmd)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to launch: %v\n", err)
		os.Exit(1)
	}
}
