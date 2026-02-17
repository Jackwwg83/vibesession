# VibeSession (`vbs`)

A TUI tool for finding, browsing, and resuming your Claude Code and Codex CLI sessions — with **automatic TTS voice output** so Claude can talk back to you.

一个用于查找、浏览和恢复 Claude Code / Codex CLI 会话的终端工具，支持 **TTS 自动语音播报**，让 Claude 的回复不只是文字。

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

- **TTS Voice Output** 🔊: Claude's responses are automatically read aloud after each reply. Supports FIFO queue for multi-session — no interruptions. 每次 Claude 回复完自动朗读，多 session 排队播放不打断。
- **Dual source**: Scans both Claude Code (`~/.claude/projects/`) and Codex CLI (`~/.codex/sessions/`)
- **TUI interface**: Searchable, filterable session list with keyboard navigation
- **Conversation viewer**: Browse the full conversation history of any session (press `v`)
- **One-step resume**: Select a session → edit the launch command → run it
- **Smart summaries**: Extracts the first user message as a readable summary
- **Fast**: Concurrent scanning, reads only the first few lines of each file

## TTS Voice Output / 语音播报

Make Claude Code talk back to you. Every time Claude finishes a reply, it's automatically read aloud.

让 Claude Code 开口说话。每次 Claude 回复完成后，自动朗读回复内容。

### Setup / 安装

```bash
# Prerequisites / 前置依赖
brew install jq
pipx install edge-tts   # or: pip3 install edge-tts

# One-command setup / 一键配置
vbs tts setup
```

### Commands / 命令

| Command | Description | 说明 |
|---------|-------------|------|
| `vbs tts setup` | First-time install: writes hook + config | 首次安装：写入 hook 和配置 |
| `vbs tts` | Show current status | 显示当前状态 |
| `vbs tts on` | Enable TTS | 开启语音 |
| `vbs tts off` | Disable TTS | 关闭语音 |
| `vbs tts next` | Skip current playback | 跳过当前播放 |
| `vbs tts clear` | Clear queue and stop | 清空队列并停止 |

### Multi-session behavior / 多会话行为

Two modes controlled by `overlap` in `~/.config/vbs/tts.json`:

通过 `~/.config/vbs/tts.json` 中的 `overlap` 字段控制：

- **`queue`** (default): Strict FIFO — current playback finishes before the next one starts. No interruptions. 严格排队，当前播完再播下一条，不打断。
- **`interrupt`**: New replies cut off the current playback immediately. 新回复立即打断当前播放。

### Config / 配置文件

`~/.config/vbs/tts.json`:
```json
{
  "enabled": true,
  "voice": "zh-CN-XiaoxiaoNeural",
  "rate": "+15%",
  "max_length": 2000,
  "overlap": "queue"
}
```

### How it works / 工作原理

1. Claude Code Stop hook triggers after each reply / 每次回复完触发 Stop hook
2. Hook extracts text, cleans markdown, writes a task to the queue / Hook 提取文本、清理 markdown、写入队列
3. A single worker process consumes tasks serially (FIFO) / 单 worker 进程串行消费（保证顺序）
4. `edge-tts` synthesizes speech, `afplay` plays it / edge-tts 合成语音，afplay 播放

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
