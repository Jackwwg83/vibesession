package tts

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Config holds TTS configuration
type Config struct {
	Enabled   bool   `json:"enabled"`
	Voice     string `json:"voice"`
	Rate      string `json:"rate"`
	MaxLength int    `json:"max_length"`
	Overlap   string `json:"overlap"`
}

var defaultConfig = Config{
	Enabled:   true,
	Voice:     "zh-CN-XiaoxiaoNeural",
	Rate:      "+15%",
	MaxLength: 2000,
	Overlap:   "queue",
}

const queueDir = "/tmp/vbs-tts-queue"
const workerPidFile = "/tmp/vbs-tts-worker.pid"

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "vbs", "tts.json")
}

func hookPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "hooks", "tts-speak.sh")
}

func workerPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "hooks", "tts-worker.sh")
}

func settingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

func readConfig() (Config, error) {
	var cfg Config
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	// backfill defaults for fields missing in old configs
	if cfg.Overlap == "" {
		cfg.Overlap = "queue"
	}
	return cfg, nil
}

func writeConfig(cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), append(data, '\n'), 0644)
}

// RunSetup performs first-time TTS installation
func RunSetup() {
	fmt.Println("🔊 VBS TTS Setup")
	fmt.Println()

	// 1. Check dependencies
	if _, err := exec.LookPath("jq"); err != nil {
		fmt.Println("[fail] jq not found")
		fmt.Println("  Install: brew install jq")
		os.Exit(1)
	}
	fmt.Println("[ok] jq found")

	if _, err := exec.LookPath("edge-tts"); err != nil {
		fmt.Println("[fail] edge-tts not found")
		fmt.Println("  Install: pip3 install edge-tts")
		os.Exit(1)
	}
	fmt.Println("[ok] edge-tts found")

	if _, err := exec.LookPath("python3"); err != nil {
		fmt.Println("[fail] python3 not found")
		os.Exit(1)
	}
	fmt.Println("[ok] python3 found")

	// 2. Write hook script + worker script
	hookDir := filepath.Dir(hookPath())
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		fmt.Printf("[fail] cannot create %s: %v\n", hookDir, err)
		os.Exit(1)
	}
	if err := os.WriteFile(hookPath(), []byte(hookScript), 0755); err != nil {
		fmt.Printf("[fail] cannot write hook script: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[ok] hook script written to %s\n", hookPath())

	if err := os.WriteFile(workerPath(), []byte(workerScript), 0755); err != nil {
		fmt.Printf("[fail] cannot write worker script: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[ok] worker script written to %s\n", workerPath())

	// 3. Update settings.json — add Stop hook
	if err := updateSettings(); err != nil {
		fmt.Printf("[fail] cannot update settings.json: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[ok] settings.json updated with Stop hook\n")

	// 4. Create default config (don't overwrite existing)
	cfgDir := filepath.Dir(configPath())
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		fmt.Printf("[fail] cannot create %s: %v\n", cfgDir, err)
		os.Exit(1)
	}
	if _, err := os.Stat(configPath()); os.IsNotExist(err) {
		if err := writeConfig(defaultConfig); err != nil {
			fmt.Printf("[fail] cannot write config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[ok] config created at %s\n", configPath())
	} else {
		fmt.Printf("[ok] config already exists at %s\n", configPath())
	}

	// 5. Create queue directory
	if err := os.MkdirAll(queueDir, 0755); err != nil {
		fmt.Printf("[fail] cannot create queue dir: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[ok] queue directory at %s\n", queueDir)

	fmt.Println()
	fmt.Println("Setup complete! TTS is now ON.")
	fmt.Println("  vbs tts        — check status")
	fmt.Println("  vbs tts off    — disable")
	fmt.Println("  vbs tts on     — enable")
	fmt.Println("  vbs tts next   — skip current, play next")
	fmt.Println("  vbs tts clear  — clear queue")
}

// RunOn enables TTS
func RunOn() {
	cfg, err := readConfig()
	if err != nil {
		fmt.Println("Config not found. Run 'vbs tts setup' first.")
		os.Exit(1)
	}
	cfg.Enabled = true
	if err := writeConfig(cfg); err != nil {
		fmt.Printf("Failed to update config: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("TTS: ON")
}

// RunOff disables TTS
func RunOff() {
	cfg, err := readConfig()
	if err != nil {
		fmt.Println("Config not found. Run 'vbs tts setup' first.")
		os.Exit(1)
	}
	cfg.Enabled = false
	if err := writeConfig(cfg); err != nil {
		fmt.Printf("Failed to update config: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("TTS: OFF")
}

// RunStatus shows current TTS configuration
func RunStatus() {
	cfg, err := readConfig()
	if err != nil {
		fmt.Println("TTS not configured. Run 'vbs tts setup' first.")
		return
	}
	if cfg.Enabled {
		fmt.Println("TTS: ON")
	} else {
		fmt.Println("TTS: OFF")
	}
	fmt.Printf("  Voice:      %s\n", cfg.Voice)
	fmt.Printf("  Rate:       %s\n", cfg.Rate)
	fmt.Printf("  Max length: %d\n", cfg.MaxLength)
	fmt.Printf("  Overlap:    %s\n", cfg.Overlap)
	fmt.Printf("  Config:     %s\n", configPath())

	// Show queue status
	entries, _ := os.ReadDir(queueDir)
	count := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			count++
		}
	}
	if count > 0 {
		fmt.Printf("  Queue:      %d pending\n", count)
	}
}

// RunNext skips the currently playing TTS and moves to the next in queue
func RunNext() {
	// Kill current afplay process
	exec.Command("pkill", "-f", "afplay /tmp/vbs-tts").Run()
	fmt.Println("Skipped current playback.")
}

// RunClear removes all pending items from the queue
func RunClear() {
	entries, err := os.ReadDir(queueDir)
	if err != nil {
		fmt.Println("Queue is empty.")
		return
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			os.Remove(filepath.Join(queueDir, e.Name()))
			count++
		}
	}
	// Also kill current playback
	exec.Command("pkill", "-f", "afplay /tmp/vbs-tts").Run()
	fmt.Printf("Cleared %d queued items and stopped playback.\n", count)
}

// updateSettings reads settings.json, adds Stop hook, writes back.
// Creates settings.json with minimal content if it doesn't exist.
func updateSettings() error {
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		if os.IsNotExist(err) {
			if mkErr := os.MkdirAll(filepath.Dir(settingsPath()), 0755); mkErr != nil {
				return fmt.Errorf("cannot create settings dir: %w", mkErr)
			}
			data = []byte("{}")
		} else {
			return fmt.Errorf("cannot read settings.json: %w", err)
		}
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("cannot parse settings.json: %w", err)
	}

	// ensure hooks map exists
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		hooks = make(map[string]interface{})
		settings["hooks"] = hooks
	}

	// build the Stop hook entry
	stopHookEntry := map[string]interface{}{
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": hookPath(),
			},
		},
	}

	// check if Stop hook already exists with our command
	if existing, ok := hooks["Stop"]; ok {
		if arr, ok := existing.([]interface{}); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					if innerHooks, ok := m["hooks"].([]interface{}); ok {
						for _, h := range innerHooks {
							if hm, ok := h.(map[string]interface{}); ok {
								if hm["command"] == hookPath() {
									return nil
								}
							}
						}
					}
				}
			}
			hooks["Stop"] = append(arr, stopHookEntry)
		} else {
			hooks["Stop"] = []interface{}{stopHookEntry}
		}
	} else {
		hooks["Stop"] = []interface{}{stopHookEntry}
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal settings.json: %w", err)
	}
	return os.WriteFile(settingsPath(), append(out, '\n'), 0644)
}

// ---------------------------------------------------------------------------
// Hook script: extracts text, enqueues task, ensures worker is running.
// No set -e. All failures are silent. Always exits 0.
// ---------------------------------------------------------------------------
var hookScript = "#!/bin/bash\n" +
	"# VBS TTS — Claude Code Stop hook\n" +
	"# Extracts assistant text -> writes queue task -> ensures worker is running\n" +
	"# Always exits 0. Never blocks Claude.\n" +
	"\n" +
	"INPUT=$(cat) || true\n" +
	"\n" +
	"CONFIG_FILE=\"$HOME/.config/vbs/tts.json\"\n" +
	"if [ ! -f \"$CONFIG_FILE\" ]; then exit 0; fi\n" +
	"\n" +
	"ENABLED=$(jq -r '.enabled' \"$CONFIG_FILE\" 2>/dev/null) || true\n" +
	"if [ \"$ENABLED\" != \"true\" ]; then exit 0; fi\n" +
	"\n" +
	"OVERLAP=$(jq -r '.overlap // \"queue\"' \"$CONFIG_FILE\" 2>/dev/null) || true\n" +
	"OVERLAP=${OVERLAP:-queue}\n" +
	"MAX_LENGTH=$(jq -r '.max_length // 2000' \"$CONFIG_FILE\" 2>/dev/null) || true\n" +
	"MAX_LENGTH=${MAX_LENGTH:-2000}\n" +
	"\n" +
	"TRANSCRIPT=$(echo \"$INPUT\" | jq -r '.transcript_path // empty' 2>/dev/null) || true\n" +
	"if [ -z \"$TRANSCRIPT\" ] || [ ! -f \"$TRANSCRIPT\" ]; then exit 0; fi\n" +
	"\n" +
	"export VBS_TRANSCRIPT=\"$TRANSCRIPT\"\n" +
	"export VBS_MAX_LENGTH=\"$MAX_LENGTH\"\n" +
	"\n" +
	"# Extract and clean text via python3\n" +
	"TEXT=$(python3 << 'PYEOF'\n" +
	"import json, sys, re, os\n" +
	"\n" +
	"transcript_path = os.environ.get(\"VBS_TRANSCRIPT\", \"\")\n" +
	"max_length = int(os.environ.get(\"VBS_MAX_LENGTH\", \"2000\"))\n" +
	"\n" +
	"if not transcript_path or not os.path.isfile(transcript_path):\n" +
	"    sys.exit(0)\n" +
	"\n" +
	"last_assistant_text = \"\"\n" +
	"with open(transcript_path, \"r\") as f:\n" +
	"    for line in f:\n" +
	"        line = line.strip()\n" +
	"        if not line:\n" +
	"            continue\n" +
	"        try:\n" +
	"            obj = json.loads(line)\n" +
	"        except json.JSONDecodeError:\n" +
	"            continue\n" +
	"        if obj.get(\"type\") == \"assistant\":\n" +
	"            message = obj.get(\"message\", {})\n" +
	"            content = message.get(\"content\", [])\n" +
	"            parts = []\n" +
	"            if isinstance(content, list):\n" +
	"                for block in content:\n" +
	"                    if isinstance(block, dict) and block.get(\"type\") == \"text\":\n" +
	"                        parts.append(block.get(\"text\", \"\"))\n" +
	"                    elif isinstance(block, str):\n" +
	"                        parts.append(block)\n" +
	"            elif isinstance(content, str):\n" +
	"                parts.append(content)\n" +
	"            text = \"\\n\".join(parts).strip()\n" +
	"            if text:\n" +
	"                last_assistant_text = text\n" +
	"\n" +
	"if not last_assistant_text:\n" +
	"    sys.exit(0)\n" +
	"\n" +
	"text = last_assistant_text\n" +
	"bt = chr(96)\n" +
	"text = re.sub(bt*3 + r'[\\s\\S]*?' + bt*3, '', text)\n" +
	"text = re.sub(bt + r'[^' + bt + r']+?' + bt, '', text)\n" +
	"text = re.sub(r'^#{1,6}\\s+', '', text, flags=re.MULTILINE)\n" +
	"text = re.sub(r'\\[([^\\]]+)\\]\\([^)]+\\)', r'\\1', text)\n" +
	"text = re.sub(r'\\*{1,3}([^*]+)\\*{1,3}', r'\\1', text)\n" +
	"text = re.sub(r'^[-*_]{3,}\\s*$', '', text, flags=re.MULTILINE)\n" +
	"text = re.sub(r'!\\[([^\\]]*)\\]\\([^)]+\\)', r'\\1', text)\n" +
	"text = re.sub(r'\\n{3,}', '\\n\\n', text)\n" +
	"text = text.strip()\n" +
	"\n" +
	"if not text:\n" +
	"    sys.exit(0)\n" +
	"\n" +
	"if len(text) > max_length:\n" +
	"    text = text[:max_length] + \"\\n\\n后面内容请查看屏幕\"\n" +
	"\n" +
	"print(text)\n" +
	"PYEOF\n" +
	") || true\n" +
	"\n" +
	"if [ -z \"$TEXT\" ]; then exit 0; fi\n" +
	"\n" +
	"QUEUEDIR=\"/tmp/vbs-tts-queue\"\n" +
	"PIDFILE=\"/tmp/vbs-tts-worker.pid\"\n" +
	"WORKER=\"$HOME/.claude/hooks/tts-worker.sh\"\n" +
	"\n" +
	"if [ \"$OVERLAP\" = \"interrupt\" ]; then\n" +
	"  # Interrupt mode: clear queue, kill worker & playback, enqueue, start fresh worker\n" +
	"  rm -f \"$QUEUEDIR\"/*.json 2>/dev/null || true\n" +
	"  pkill -f \"afplay /tmp/vbs-tts\" 2>/dev/null || true\n" +
	"  # Kill existing worker\n" +
	"  if [ -f \"$PIDFILE\" ]; then\n" +
	"    OLD_PID=$(cat \"$PIDFILE\" 2>/dev/null)\n" +
	"    if [ -n \"$OLD_PID\" ]; then kill \"$OLD_PID\" 2>/dev/null || true; fi\n" +
	"    rm -f \"$PIDFILE\"\n" +
	"  fi\n" +
	"fi\n" +
	"\n" +
	"# Enqueue: write task JSON with timestamp-based filename for FIFO ordering\n" +
	"mkdir -p \"$QUEUEDIR\" 2>/dev/null || true\n" +
	"TASK_ID=$(python3 -c 'import time; print(f\"{time.time_ns()}\")' 2>/dev/null || date +%s)_$$\n" +
	"TASK_FILE=\"$QUEUEDIR/${TASK_ID}.json\"\n" +
	"\n" +
	"# Write task as JSON (text in a file to avoid escaping issues)\n" +
	"TXTFILE=\"/tmp/vbs-tts-text-${TASK_ID}.txt\"\n" +
	"printf '%s' \"$TEXT\" > \"$TXTFILE\" 2>/dev/null || true\n" +
	"\n" +
	"cat > \"$TASK_FILE\" 2>/dev/null << TASKEOF\n" +
	"{\"text_file\": \"$TXTFILE\", \"created\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}\n" +
	"TASKEOF\n" +
	"\n" +
	"# Ensure worker is running\n" +
	"WORKER_ALIVE=false\n" +
	"if [ -f \"$PIDFILE\" ]; then\n" +
	"  OLD_PID=$(cat \"$PIDFILE\" 2>/dev/null)\n" +
	"  if [ -n \"$OLD_PID\" ] && kill -0 \"$OLD_PID\" 2>/dev/null; then\n" +
	"    WORKER_ALIVE=true\n" +
	"  fi\n" +
	"fi\n" +
	"\n" +
	"if [ \"$WORKER_ALIVE\" = \"false\" ]; then\n" +
	"  nohup \"$WORKER\" > /dev/null 2>&1 &\n" +
	"fi\n" +
	"\n" +
	"exit 0\n"

// ---------------------------------------------------------------------------
// Worker script: single process that serially consumes the queue.
// Picks oldest task -> synthesizes -> plays -> deletes -> loops.
// Exits when queue is empty for 2 seconds (will be re-spawned by next hook).
// ---------------------------------------------------------------------------
var workerScript = "#!/bin/bash\n" +
	"# VBS TTS Worker — serial queue consumer\n" +
	"# Spawned by hook. Processes tasks FIFO. Exits when idle.\n" +
	"\n" +
	"QUEUEDIR=\"/tmp/vbs-tts-queue\"\n" +
	"PIDFILE=\"/tmp/vbs-tts-worker.pid\"\n" +
	"CONFIG_FILE=\"$HOME/.config/vbs/tts.json\"\n" +
	"\n" +
	"# Write our PID\n" +
	"echo $$ > \"$PIDFILE\"\n" +
	"trap 'rm -f \"$PIDFILE\"' EXIT\n" +
	"\n" +
	"IDLE_COUNT=0\n" +
	"\n" +
	"while true; do\n" +
	"  # Find oldest task (sorted by filename = timestamp)\n" +
	"  TASK=$(ls \"$QUEUEDIR\"/*.json 2>/dev/null | sort | head -1)\n" +
	"\n" +
	"  if [ -z \"$TASK\" ]; then\n" +
	"    IDLE_COUNT=$((IDLE_COUNT + 1))\n" +
	"    # Exit after 2 seconds of no tasks (4 x 0.5s)\n" +
	"    if [ \"$IDLE_COUNT\" -ge 4 ]; then\n" +
	"      break\n" +
	"    fi\n" +
	"    sleep 0.5\n" +
	"    continue\n" +
	"  fi\n" +
	"\n" +
	"  IDLE_COUNT=0\n" +
	"\n" +
	"  # Read task\n" +
	"  TXTFILE=$(jq -r '.text_file // empty' \"$TASK\" 2>/dev/null) || true\n" +
	"\n" +
	"  # Remove task from queue immediately (claimed)\n" +
	"  rm -f \"$TASK\"\n" +
	"\n" +
	"  if [ -z \"$TXTFILE\" ] || [ ! -f \"$TXTFILE\" ]; then\n" +
	"    continue\n" +
	"  fi\n" +
	"\n" +
	"  # Read config for voice/rate (re-read each time so changes take effect)\n" +
	"  VOICE=$(jq -r '.voice // \"zh-CN-XiaoxiaoNeural\"' \"$CONFIG_FILE\" 2>/dev/null) || true\n" +
	"  RATE=$(jq -r '.rate // \"+15%\"' \"$CONFIG_FILE\" 2>/dev/null) || true\n" +
	"  VOICE=${VOICE:-zh-CN-XiaoxiaoNeural}\n" +
	"  RATE=${RATE:-+15%}\n" +
	"\n" +
	"  # Synthesize\n" +
	"  MPFILE=\"/tmp/vbs-tts-$$.mp3\"\n" +
	"  edge-tts --voice \"$VOICE\" --rate \"$RATE\" -f \"$TXTFILE\" --write-media \"$MPFILE\" 2>/dev/null || true\n" +
	"\n" +
	"  # Play\n" +
	"  if [ -f \"$MPFILE\" ]; then\n" +
	"    afplay \"$MPFILE\" 2>/dev/null || true\n" +
	"  fi\n" +
	"\n" +
	"  # Cleanup\n" +
	"  rm -f \"$MPFILE\" \"$TXTFILE\"\n" +
	"done\n" +
	"\n" +
	"rm -f \"$PIDFILE\"\n"
