package model

// Message represents a parsed conversation message from a session file.
type Message struct {
	Role      string   // "user" or "assistant"
	Text      string   // rendered message content
	ToolCalls []string // tool call summaries like "Read: main.go"
	Index     int      // sequential index in conversation
}
