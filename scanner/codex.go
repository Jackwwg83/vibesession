package scanner

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackwu/vibesession/model"
)

func ScanCodex() []model.Session {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	sessionsDir := filepath.Join(homeDir, ".codex", "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return nil
	}

	var sessions []model.Session

	err = filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".jsonl") {
			return nil
		}

		s := parseCodexSession(path, info)
		if s != nil {
			sessions = append(sessions, *s)
		}
		return nil
	})

	if err != nil {
		return nil
	}

	return sessions
}

func parseCodexSession(filePath string, info os.FileInfo) *model.Session {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)

	var sessionID string
	var cwd string
	var summary string

	for i := 0; i < 50 && scanner.Scan(); i++ {
		var line map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}

		lineType, _ := line["type"].(string)

		// extract metadata from session_meta
		if lineType == "session_meta" {
			if payload, ok := line["payload"].(map[string]interface{}); ok {
				sessionID, _ = payload["id"].(string)
				cwd, _ = payload["cwd"].(string)
			}
		}

		// extract first real user message from response_item
		if summary == "" && lineType == "response_item" {
			if payload, ok := line["payload"].(map[string]interface{}); ok {
				role, _ := payload["role"].(string)
				if role == "user" {
					text := extractCodexText(payload)
					// skip system-like messages
					if text != "" &&
						!strings.Contains(text, "<environment_context>") &&
						!strings.Contains(text, "AGENTS.md") &&
						!strings.Contains(text, "<permissions") &&
						!strings.HasPrefix(text, "#") {
						summary = text
					}
				}
			}
		}

		if sessionID != "" && summary != "" {
			break
		}
	}

	if sessionID == "" {
		return nil
	}

	summary = truncate(summary, 120)
	if summary == "" {
		summary = "(no message)"
	}

	project := filepath.Base(cwd)
	if project == "" || project == "." {
		project = "unknown"
	}

	return &model.Session{
		ID:       sessionID,
		ShortID:  shortID(sessionID),
		Source:   model.SourceCodex,
		Time:     info.ModTime(),
		Project:  project,
		CWD:      cwd,
		Summary:  summary,
		FilePath: filePath,
	}
}

// extractCodexText extracts the text from a Codex response_item payload.
// Content is an array of objects with "type" and "text" fields.
func extractCodexText(payload map[string]interface{}) string {
	content, ok := payload["content"].([]interface{})
	if !ok {
		return ""
	}

	for _, item := range content {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		itemType, _ := obj["type"].(string)
		if itemType == "input_text" {
			text, _ := obj["text"].(string)
			if text != "" {
				return text
			}
		}
	}
	return ""
}
