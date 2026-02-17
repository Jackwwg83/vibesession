package scanner

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jackwu/vibesession/model"
)

var teammateTagRe = regexp.MustCompile(`<teammate-message[^>]*>`)
var teammateCloseRe = regexp.MustCompile(`</teammate-message>`)

func ScanClaude() []model.Session {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	projectsDir := filepath.Join(homeDir, ".claude", "projects")
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		return nil
	}

	var sessions []model.Session

	projectEntries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil
	}

	for _, projEntry := range projectEntries {
		if !projEntry.IsDir() {
			continue
		}
		projPath := filepath.Join(projectsDir, projEntry.Name())
		fileEntries, err := os.ReadDir(projPath)
		if err != nil {
			continue
		}

		for _, fe := range fileEntries {
			name := fe.Name()
			// only top-level .jsonl files, skip directories (subagents)
			if fe.IsDir() || !strings.HasSuffix(name, ".jsonl") {
				continue
			}

			filePath := filepath.Join(projPath, name)
			s := parseClaudeSession(filePath)
			if s != nil {
				sessions = append(sessions, *s)
			}
		}
	}

	return sessions
}

func parseClaudeSession(filePath string) *model.Session {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil
	}

	scanner := bufio.NewScanner(f)
	// increase buffer for potentially large first lines
	scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)

	// scan lines to find the first one with a sessionId
	// (some files start with file-history-snapshot or other non-session lines)
	var firstLine struct {
		SessionID string `json:"sessionId"`
		CWD       string `json:"cwd"`
		Type      string `json:"type"`
		TeamName  string `json:"teamName"`
		Message   struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	}

	found := false
	for i := 0; i < 10 && scanner.Scan(); i++ {
		if err := json.Unmarshal(scanner.Bytes(), &firstLine); err != nil {
			continue
		}
		if firstLine.SessionID != "" {
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	summary := firstLine.Message.Content
	// strip teammate message tags
	summary = teammateTagRe.ReplaceAllString(summary, "")
	summary = teammateCloseRe.ReplaceAllString(summary, "")
	summary = strings.TrimSpace(summary)

	// for teammate sessions, extract the task description instead of boilerplate
	if strings.HasPrefix(summary, "You are") {
		// try to find the actual task content after "Your task is"
		if idx := strings.Index(summary, "Your task is"); idx >= 0 {
			summary = summary[idx:]
		}
	}
	// strip XML-like system tags
	summary = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(summary, "")
	summary = strings.TrimSpace(summary)

	// if first line is not a user message, scan ahead
	if firstLine.Type != "user" || summary == "" {
		for i := 0; i < 20 && scanner.Scan(); i++ {
			var line struct {
				Type    string `json:"type"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
				continue
			}
			if line.Type == "user" && line.Message.Content != "" {
				summary = line.Message.Content
				summary = teammateTagRe.ReplaceAllString(summary, "")
				summary = teammateCloseRe.ReplaceAllString(summary, "")
				summary = strings.TrimSpace(summary)
				break
			}
		}
	}

	// truncate summary
	summary = truncate(summary, 120)

	// derive project name from CWD
	project := filepath.Base(firstLine.CWD)
	if project == "" || project == "." {
		project = "unknown"
	}

	return &model.Session{
		ID:       firstLine.SessionID,
		ShortID:  shortID(firstLine.SessionID),
		Source:   model.SourceClaude,
		Time:     info.ModTime(),
		Project:  project,
		CWD:      firstLine.CWD,
		Summary:  summary,
		FilePath: filePath,
		TeamName: firstLine.TeamName,
	}
}

func shortID(id string) string {
	if len(id) < 9 {
		return id
	}
	return id[:4] + ".." + id[len(id)-4:]
}

func truncate(s string, maxLen int) string {
	// replace newlines with spaces
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	// collapse multiple spaces
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	s = strings.TrimSpace(s)

	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-2]) + ".."
}
