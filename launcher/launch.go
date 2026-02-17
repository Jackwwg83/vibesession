package launcher

import (
	"fmt"

	"github.com/jackwu/vibesession/model"
)

// BuildCommand returns the shell command to resume a session.
func BuildCommand(s model.Session) string {
	switch s.Source {
	case model.SourceClaude:
		return fmt.Sprintf("cd %s && claude -r %s", shellQuote(s.CWD), s.ID)
	case model.SourceCodex:
		return fmt.Sprintf("cd %s && codex resume %s", shellQuote(s.CWD), s.ID)
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
