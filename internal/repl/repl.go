// Package repl provides the interactive read-eval-print loop for the dojo CLI.
package repl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/DojoGenesis/dojo-cli/internal/client"
	"github.com/DojoGenesis/dojo-cli/internal/commands"
	"github.com/DojoGenesis/dojo-cli/internal/config"
	"github.com/DojoGenesis/dojo-cli/internal/hooks"
	"github.com/DojoGenesis/dojo-cli/internal/plugins"
	"github.com/chzyer/readline"
	"github.com/fatih/color"
)

// REPL is the interactive session.
type REPL struct {
	cfg      *config.Config
	gw       *client.Client
	registry *commands.Registry
	runner   *hooks.Runner
	session  string // active session ID
}

// New creates a REPL bound to the given config and gateway client.
// It scans cfg.Plugins.Path for CoworkPlugins-format directories on startup.
// If scanning fails a warning is logged and the REPL continues with no hooks.
func New(cfg *config.Config, gw *client.Client) *REPL {
	plgs, err := plugins.Scan(cfg.Plugins.Path)
	if err != nil {
		log.Printf("[repl] warning: plugin scan failed (%s): %v — continuing with no plugins", cfg.Plugins.Path, err)
		plgs = nil
	}
	if len(plgs) > 0 {
		log.Printf("[repl] loaded %d plugin(s) from %s", len(plgs), cfg.Plugins.Path)
	}

	session := fmt.Sprintf("dojo-cli-%s", time.Now().Format("20060102-150405"))
	reg := commands.New(cfg, gw, plgs)
	return &REPL{
		cfg:      cfg,
		gw:       gw,
		session:  session,
		registry: reg,
		runner:   reg.Runner(),
	}
}

// Run starts the interactive loop. Returns when the user exits.
func (r *REPL) Run(ctx context.Context) error {
	printWelcome(r.cfg, r.session)

	rl, err := newReadline()
	if err != nil {
		// Fallback to plain stdin if readline init fails (e.g. in pipes)
		return r.runPlain(ctx)
	}
	defer rl.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				fmt.Println()
				continue
			}
			if err == io.EOF {
				fmt.Println("\ngoodbye")
				return nil
			}
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" || line == "bye" {
			fmt.Println("\ngoodbye")
			return nil
		}

		if err := r.handle(ctx, line); err != nil {
			color.Red("  error: %s\n", err)
		}
	}
}

// handle routes a line to either a slash command or a chat message.
// For slash commands it fires PreCommand before dispatch and PostCommand after
// a successful dispatch. Hook errors are logged but not fatal.
func (r *REPL) handle(ctx context.Context, line string) error {
	if strings.HasPrefix(line, "/") {
		payload := map[string]any{"command": line}

		if err := r.runner.Fire(ctx, hooks.EventPreCommand, payload); err != nil {
			log.Printf("[hooks] PreCommand error: %v", err)
		}

		cmdErr := r.registry.Dispatch(ctx, line[1:])

		if cmdErr == nil {
			if err := r.runner.Fire(ctx, hooks.EventPostCommand, payload); err != nil {
				log.Printf("[hooks] PostCommand error: %v", err)
			}
		}
		return cmdErr
	}
	return r.chat(ctx, line)
}

// chat sends a freeform message to the gateway and streams the response.
func (r *REPL) chat(ctx context.Context, message string) error {
	req := client.ChatRequest{
		Message:   message,
		Model:     r.cfg.Defaults.Model,
		SessionID: r.session,
		Stream:    true,
	}

	fmt.Println()
	prefix := color.New(color.FgGreen, color.Bold)
	prefix.Print("  dojo  ")

	var fullText strings.Builder
	err := r.gw.ChatStream(ctx, req, func(chunk client.SSEChunk) {
		text := extractText(chunk)
		if text != "" {
			fmt.Print(text)
			fullText.WriteString(text)
		}
	})

	fmt.Println()
	fmt.Println()
	return err
}

// extractText pulls the readable text from an SSE chunk.
// The gateway may send raw text, or a JSON object with a "text"/"content" field.
func extractText(chunk client.SSEChunk) string {
	data := strings.TrimSpace(chunk.Data)
	if data == "" || data == "[DONE]" {
		return ""
	}

	// Try JSON unwrap
	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err == nil {
		// OpenAI delta format
		if choices, ok := m["choices"].([]any); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]any); ok {
				if delta, ok := choice["delta"].(map[string]any); ok {
					if content, ok := delta["content"].(string); ok {
						return content
					}
				}
				// non-streaming text field
				if text, ok := choice["text"].(string); ok {
					return text
				}
			}
		}
		// Simple {"text": "..."} or {"content": "..."}
		for _, key := range []string{"text", "content", "message", "response"} {
			if v, ok := m[key].(string); ok {
				return v
			}
		}
		return ""
	}

	// Plain text chunk
	return data
}

// runPlain is the fallback when readline is unavailable (piped input, CI).
func (r *REPL) runPlain(ctx context.Context) error {
	printWelcome(r.cfg, r.session)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			return nil
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return nil
		}
		if err := r.handle(ctx, line); err != nil {
			color.Red("error: %s\n", err)
		}
	}
}

// ─── readline setup ──────────────────────────────────────────────────────────

func newReadline() (*readline.Instance, error) {
	completer := readline.NewPrefixCompleter(
		readline.PcItem("/help"),
		readline.PcItem("/health"),
		readline.PcItem("/home"),
		readline.PcItem("/model"),
		readline.PcItem("/tools"),
		readline.PcItem("/agent", readline.PcItem("ls")),
		readline.PcItem("/skill", readline.PcItem("ls")),
		readline.PcItem("/garden",
			readline.PcItem("ls"),
			readline.PcItem("stats"),
			readline.PcItem("plant"),
			readline.PcItem("harvest"),
		),
		readline.PcItem("/trail"),
		readline.PcItem("/trace"),
		readline.PcItem("/pilot"),
		readline.PcItem("/hooks",
			readline.PcItem("ls"),
			readline.PcItem("fire"),
		),
		readline.PcItem("/settings"),
		readline.PcItem("exit"),
	)

	return readline.NewEx(&readline.Config{
		Prompt:          color.GreenString("dojo") + color.HiBlackString(" › "),
		HistoryFile:     historyPath(),
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
}

func historyPath() string {
	home, _ := os.UserHomeDir()
	return home + "/.dojo/.history"
}

// ─── Welcome banner ──────────────────────────────────────────────────────────

func printWelcome(cfg *config.Config, session string) {
	green := color.New(color.FgGreen, color.Bold)
	dim := color.New(color.FgHiBlack)
	fmt.Println()
	green.Println("  Dojo CLI")
	dim.Printf("  gateway: %s\n", cfg.Gateway.URL)
	dim.Printf("  session: %s\n", session)
	dim.Println("  type /help for commands, /health to check the gateway")
	fmt.Println()
}
