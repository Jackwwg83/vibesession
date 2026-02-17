package model

import "time"

type Source string

const (
	SourceClaude Source = "Claude"
	SourceCodex  Source = "Codex"
)

type Session struct {
	ID       string
	ShortID  string // first4..last4
	Source   Source
	Time     time.Time
	Project  string // last component of CWD
	CWD      string // full working directory path
	Summary  string // first user message, truncated
	FilePath string // path to .jsonl file
}
