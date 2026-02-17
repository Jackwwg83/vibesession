package launcher

import (
	"fmt"

	"github.com/jackwu/vibesession/model"
)

// BuildCommand returns the shell command to resume a session.
func BuildCommand(s model.Session) string {
	switch s.Source {
	case model.SourceClaude:
		return fmt.Sprintf("cd %s && claude -r %s", shellQuote(s.CWD), shellQuote(s.ID))
	case model.SourceCodex:
		return fmt.Sprintf("cd %s && codex resume %s", shellQuote(s.CWD), shellQuote(s.ID))
	default:
		return ""
	}
}

// BuildYoloCommand returns the shell command to resume a session in yolo mode.
func BuildYoloCommand(s model.Session) string {
	switch s.Source {
	case model.SourceClaude:
		return fmt.Sprintf("cd %s && claude -r %s --dangerously-skip-permissions", shellQuote(s.CWD), shellQuote(s.ID))
	case model.SourceCodex:
		return fmt.Sprintf("cd %s && codex resume %s --full-auto", shellQuote(s.CWD), shellQuote(s.ID))
	default:
		return ""
	}
}

// BuildNewCommand returns the shell command to start a new session.
func BuildNewCommand(tool string, dir string, yolo bool) string {
	cd := fmt.Sprintf("cd %s", shellQuote(dir))
	switch tool {
	case "claude":
		if yolo {
			return cd + " && claude --dangerously-skip-permissions"
		}
		return cd + " && claude"
	case "codex":
		if yolo {
			return cd + " && codex --full-auto"
		}
		return cd + " && codex"
	default:
		return ""
	}
}

func shellQuote(s string) string {
	// simple quoting: wrap in single quotes, escape existing single quotes
	return "'" + replaceAll(s, "'", "'\\''") + "'"
}

func replaceAll(s, old, new string) string {
	result := ""
	for {
		i := indexOf(s, old)
		if i < 0 {
			return result + s
		}
		result += s[:i] + new
		s = s[i+len(old):]
	}
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
