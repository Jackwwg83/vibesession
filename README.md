# VibeSession (`vbs`)

A TUI tool for finding, browsing, and resuming your Claude Code and Codex CLI sessions.

If you run multiple AI coding sessions across different projects and terminals, you know the pain — dozens of sessions with UUID names scattered across your filesystem, and no way to find "that one where I was debugging the payment API last Tuesday."

`vbs` scans all your Claude Code and Codex sessions, shows them in a searchable list, and lets you browse full conversation history or jump back into any session with two keystrokes.

## Quick Start

```bash
# Install (macOS Apple Silicon — see Install section for other platforms)
curl -L -o vbs https://github.com/Jackwwg83/vibesession/releases/latest/download/vbs-darwin-arm64
chmod +x vbs
mkdir -p ~/bin && mv vbs ~/bin/

# Verify it's in your PATH
command -v vbs || echo 'Add ~/bin to your PATH: export PATH="$HOME/bin:$PATH"'

# Run
vbs              # open TUI
vbs --list       # plain text list (for scripting)
```

## Features

- **Dual source**: Scans both Claude Code (`~/.claude/projects/`) and Codex CLI (`~/.codex/sessions/`)
- **TUI interface**: Searchable, filterable session list with keyboard navigation
- **Conversation viewer**: Browse the full conversation history of any session (press `v`)
- **One-step resume**: Select a session → edit the launch command → run it
- **Smart summaries**: Extracts the first user message as a readable summary
- **Fast**: Concurrent scanning, reads only the first few lines of each file

## Usage

### Session List (main screen)

| Key | Action |
|-----|--------|
| `↑↓` / `j/k` | Navigate sessions |
| `Enter` | Show editable launch command |
| `v` | View full conversation history |
| `/` | Search (matches project, summary, session ID) |
| `Tab` | Filter: All → Claude → Codex |
| `PgUp/PgDn` | Scroll fast |
| `g` / `G` | Jump to top / bottom |
| `q` | Quit |

### Conversation Detail (press `v`)

| Key | Action |
|-----|--------|
| `↑↓` / `j/k` | Scroll |
| `d` / `u` | Page down / up |
| `g` / `G` | Jump to top / bottom |
| `/` | Search within conversation |
| `n` / `N` | Next / previous search match |
| `Enter` | Launch this session |
| `Esc` / `q` | Back to session list |

### Command Edit (press `Enter`)

| Key | Action |
|-----|--------|
| `Enter` | Execute the command (resumes session) |
| `Esc` | Cancel, return to previous view |

You can edit the command before executing — add flags like `--yolo`, change directory, etc.

### Launch Flow

1. Select a session in the list
2. Press `Enter` — an editable command appears:
   ```
   > cd '/Users/you/project' && claude -r abc-123
   ```
3. Edit if needed, then press `Enter` to run

## Install

### From source (requires Go 1.21+)

```bash
git clone https://github.com/Jackwwg83/vibesession.git
cd vibesession
go build -o vbs .
mkdir -p ~/bin && cp vbs ~/bin/
```

Make sure `~/bin` is in your PATH:

```bash
# Add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/bin:$PATH"
```

### From release (no Go required)

Download the pre-built binary from [Releases](https://github.com/Jackwwg83/vibesession/releases):

```bash
# macOS Apple Silicon
curl -L -o vbs https://github.com/Jackwwg83/vibesession/releases/latest/download/vbs-darwin-arm64

# macOS Intel
curl -L -o vbs https://github.com/Jackwwg83/vibesession/releases/latest/download/vbs-darwin-amd64

# Linux x86_64
curl -L -o vbs https://github.com/Jackwwg83/vibesession/releases/latest/download/vbs-linux-amd64
```

Then install:

```bash
chmod +x vbs
mkdir -p ~/bin && mv vbs ~/bin/
```

## How It Works

- **Claude Code**: Scans `~/.claude/projects/*/` for `.jsonl` transcript files. Parses the first few lines for session ID, working directory, and first user message. The conversation viewer reads the full file to display all user/assistant exchanges and tool call summaries.
- **Codex CLI**: Scans `~/.codex/sessions/YYYY/MM/DD/` for `.jsonl` session files. Parses `session_meta` for metadata and extracts messages from `response_item` entries.

No data is modified. `vbs` is read-only.

## Troubleshooting

**"No sessions found"**
- Check that session directories exist: `ls ~/.claude/projects/` and/or `ls ~/.codex/sessions/`
- You need at least one past Claude Code or Codex CLI session

**`command not found: vbs`**
- Ensure `~/bin` is in your PATH: `echo $PATH | grep -q "$HOME/bin" && echo OK || echo "Add ~/bin to PATH"`

**Conversation viewer shows "(parse stopped: encountered an oversized line)"**
- Some sessions contain very large tool outputs. The viewer handles up to 10MB per line; anything larger is skipped with a warning.

## Requirements

- macOS or Linux
- Claude Code and/or Codex CLI installed (at least one)

## License

MIT
