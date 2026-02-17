package scanner

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jackwu/vibesession/model"
)

// ParseMessages reads a session JSONL file and returns parsed conversation messages.
func ParseMessages(filePath string, source model.Source) []model.Message {
	switch source {
	case model.SourceClaude:
		return parseClaudeMessages(filePath)
	case model.SourceCodex:
		return parseCodexMessages(filePath)
	default:
		return nil
	}
}

func parseClaudeMessages(filePath string) []model.Message {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 256*1024), 10*1024*1024) // 10MB to handle large tool outputs

	var messages []model.Message
	idx := 0

	for sc.Scan() {
		var line struct {
			Type    string `json:"type"`
			Message struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			continue
		}

		switch line.Type {
		case "user":
			text, isToolResult := extractClaudeUserContent(line.Message.Content)
			if isToolResult || text == "" {
				continue
			}
			messages = append(messages, model.Message{
				Role:  "user",
				Text:  text,
				Index: idx,
			})
			idx++

		case "assistant":
			text, tools := extractClaudeAssistantContent(line.Message.Content)
			if text == "" && len(tools) == 0 {
				continue
			}
			// merge with previous assistant message if exists
			if len(messages) > 0 && messages[len(messages)-1].Role == "assistant" {
				prev := &messages[len(messages)-1]
				if text != "" {
					if prev.Text != "" {
						prev.Text += "\n" + text
					} else {
						prev.Text = text
					}
				}
				prev.ToolCalls = append(prev.ToolCalls, tools...)
				continue
			}
			messages = append(messages, model.Message{
				Role:      "assistant",
				Text:      text,
				ToolCalls: tools,
				Index:     idx,
			})
			idx++
		}
	}

	if err := sc.Err(); err != nil {
		hint := "(parse error: some messages may be missing)"
		if errors.Is(err, bufio.ErrTooLong) {
			hint = "(parse stopped: encountered an oversized line)"
		}
		messages = append(messages, model.Message{
			Role:  "assistant",
			Text:  hint,
			Index: idx,
		})
	}

	return messages
}

// extractClaudeUserContent returns the text and whether this is a tool_result.
func extractClaudeUserContent(raw json.RawMessage) (string, bool) {
	// content can be a plain string
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		if isClaudeSystemContent(str) {
			return "", false
		}
		return str, false
	}

	// or an array of content blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		hasToolResult := false
		var texts []string
		for _, b := range blocks {
			if b.Type == "tool_result" {
				hasToolResult = true
			}
			if b.Type == "text" && b.Text != "" && !isClaudeSystemContent(b.Text) {
				texts = append(texts, b.Text)
			}
		}
		if len(texts) > 0 {
			return strings.Join(texts, "\n"), false
		}
		if hasToolResult {
			return "", true
		}
	}

	return "", false
}

// isClaudeSystemContent returns true for system-generated messages that should be skipped.
func isClaudeSystemContent(text string) bool {
	return strings.HasPrefix(text, "<local-command-") ||
		strings.HasPrefix(text, "<command-name>") ||
		strings.HasPrefix(text, "<local-command-stdout>") ||
		strings.HasPrefix(text, "<local-command-caveat>") ||
		strings.Contains(text, "<system-reminder>") ||
		strings.HasPrefix(text, "<environment_context>")
}

// extractClaudeAssistantContent extracts text and tool calls from assistant content blocks.
func extractClaudeAssistantContent(raw json.RawMessage) (string, []string) {
	var blocks []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return "", nil
	}

	var texts []string
	var tools []string

	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				texts = append(texts, b.Text)
			}
		case "tool_use":
			summary := formatToolCall(b.Name, b.Input)
			tools = append(tools, summary)
		// skip "thinking" blocks
		}
	}

	return strings.Join(texts, "\n"), tools
}

// formatToolCall creates a short summary of a tool call.
func formatToolCall(name string, input json.RawMessage) string {
	var params map[string]interface{}
	if err := json.Unmarshal(input, &params); err != nil {
		return name
	}

	switch name {
	case "Read":
		if fp, ok := params["file_path"].(string); ok {
			return fmt.Sprintf("Read: %s", shortPath(fp))
		}
	case "Write":
		if fp, ok := params["file_path"].(string); ok {
			return fmt.Sprintf("Write: %s", shortPath(fp))
		}
	case "Edit":
		if fp, ok := params["file_path"].(string); ok {
			return fmt.Sprintf("Edit: %s", shortPath(fp))
		}
	case "Glob":
		if p, ok := params["pattern"].(string); ok {
			return fmt.Sprintf("Glob: %s", p)
		}
	case "Grep":
		if p, ok := params["pattern"].(string); ok {
			return fmt.Sprintf("Grep: %s", truncateStr(p, 40))
		}
	case "Bash":
		if cmd, ok := params["command"].(string); ok {
			return fmt.Sprintf("Bash: %s", truncateStr(cmd, 60))
		}
	case "WebSearch":
		if q, ok := params["query"].(string); ok {
			return fmt.Sprintf("WebSearch: %s", truncateStr(q, 50))
		}
	case "WebFetch":
		if u, ok := params["url"].(string); ok {
			return fmt.Sprintf("WebFetch: %s", truncateStr(u, 60))
		}
	case "Task":
		if desc, ok := params["description"].(string); ok {
			return fmt.Sprintf("Task: %s", truncateStr(desc, 50))
		}
	}

	// generic: show first string param
	for _, v := range params {
		if s, ok := v.(string); ok && s != "" {
			return fmt.Sprintf("%s: %s", name, truncateStr(s, 50))
		}
	}
	return name
}

func shortPath(p string) string {
	parts := strings.Split(p, "/")
	if len(parts) <= 3 {
		return p
	}
	return strings.Join(parts[len(parts)-3:], "/")
}

func truncateStr(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-2]) + ".."
}

func parseCodexMessages(filePath string) []model.Message {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 256*1024), 10*1024*1024) // 10MB to handle large tool outputs

	var messages []model.Message
	idx := 0

	for sc.Scan() {
		var line struct {
			Type    string `json:"type"`
			Payload struct {
				Type    string `json:"type"`
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			continue
		}

		if line.Type != "response_item" || line.Payload.Type != "message" {
			continue
		}

		role := line.Payload.Role
		if role != "user" && role != "assistant" {
			continue
		}

		var text string
		for _, c := range line.Payload.Content {
			switch c.Type {
			case "input_text", "output_text":
				if c.Text != "" {
					// skip system-like messages
					if role == "user" && isCodexSystemMessage(c.Text) {
						continue
					}
					if text != "" {
						text += "\n"
					}
					text += c.Text
				}
			}
		}

		if text == "" {
			continue
		}

		// merge consecutive assistant messages
		if role == "assistant" && len(messages) > 0 && messages[len(messages)-1].Role == "assistant" {
			prev := &messages[len(messages)-1]
			prev.Text += "\n" + text
			continue
		}

		messages = append(messages, model.Message{
			Role:  role,
			Text:  text,
			Index: idx,
		})
		idx++
	}

	if err := sc.Err(); err != nil {
		hint := "(parse error: some messages may be missing)"
		if errors.Is(err, bufio.ErrTooLong) {
			hint = "(parse stopped: encountered an oversized line)"
		}
		messages = append(messages, model.Message{
			Role:  "assistant",
			Text:  hint,
			Index: idx,
		})
	}

	return messages
}

func isCodexSystemMessage(text string) bool {
	return strings.Contains(text, "<environment_context>") ||
		strings.Contains(text, "AGENTS.md") ||
		strings.Contains(text, "<permissions")
}
