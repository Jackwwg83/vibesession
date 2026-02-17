# VibeSession (`vbs`)

A TUI tool for finding and resuming your Claude Code and Codex CLI sessions.

If you run multiple AI coding sessions across different projects and terminals, you know the pain — dozens of sessions with UUID names scattered across your filesystem, and no way to find "that one where I was debugging the payment API last Tuesday."

`vbs` scans all your Claude Code and Codex sessions, shows them in a searchable list, and lets you jump back into any session with two keystrokes.

## Features

- **Dual source**: Scans both Claude Code (`~/.claude/projects/`) and Codex CLI (`~/.codex/sessions/`)
- **TUI interface**: Searchable, filterable session list with keyboard navigation
- **One-step resume**: Select a session → edit the launch command → run it
- **Smart summaries**: Extracts the first user message as a readable summary
- **Fast**: Concurrent scanning, reads only the first few lines of each file

## Install

### From source (requires Go 1.21+)

```bash
git clone https://github.com/Jackwwg83/vibesession.git
cd vibesession
go build -o vbs .
cp vbs ~/bin/  # or /usr/local/bin/
```

### From release

Download the binary from [Releases](https://github.com/Jackwwg83/vibesession/releases) and add it to your PATH.

## Usage

```bash
# Open TUI
vbs

# Plain text list (for scripting)
vbs --list
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑↓` / `j/k` | Navigate |
| `Enter` | Show editable launch command → `Enter` again to execute |
| `/` | Search (matches project, summary, session ID) |
| `Tab` | Filter: All → Claude → Codex |
| `PgUp/PgDn` | Scroll fast |
| `Esc` | Cancel / back |
| `q` | Quit |

### Launch Flow

1. Select a session
2. Press `Enter` — an editable command appears at the bottom:
   ```
   > cd '/Users/you/project' && claude -r abc-123
   ```
3. Add flags if needed (e.g. `--yolo`) or just press `Enter` to run

## How It Works

- **Claude Code**: Scans `~/.claude/projects/*/` for `.jsonl` transcript files. Parses the first line for session ID, working directory, and first user message.
- **Codex CLI**: Scans `~/.codex/sessions/YYYY/MM/DD/` for `.jsonl` session files. Parses `session_meta` for metadata and finds the first `user` message in `response_item` entries.

No data is modified. `vbs` is read-only.

## Requirements

- macOS or Linux
- Claude Code and/or Codex CLI installed (at least one)

## License

MIT
