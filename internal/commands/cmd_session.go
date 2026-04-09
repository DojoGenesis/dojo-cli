package commands

// cmd_session.go — /session command.

import (
	"context"
	"fmt"
	"time"

	"github.com/DojoGenesis/dojo-cli/internal/state"
	gcolor "github.com/gookit/color"
)

// ─── /session ───────────────────────────────────────────────────────────────

func (r *Registry) sessionCmd() Command {
	return Command{
		Name:  "session",
		Usage: "/session [new|resume|<id>]",
		Short: "Show or change the active session ID",
		Run: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				fmt.Println()
				printKV("session", *r.session)
				fmt.Println()
				return nil
			}
			switch args[0] {
			case "new":
				*r.session = fmt.Sprintf("dojo-cli-%s", time.Now().Format("20060102-150405"))
				fmt.Println()
				fmt.Println(gcolor.HEX("#7fb88c").Sprint("  New session started"))
				printKV("session", *r.session)
				fmt.Println()
				state.SaveSession(*r.session)
			case "resume":
				st, err := state.Load()
				if err != nil || st.LastSessionID == "" {
					return fmt.Errorf("no prior session to resume")
				}
				*r.session = st.LastSessionID
				fmt.Println()
				fmt.Println(gcolor.HEX("#7fb88c").Sprint("  Session resumed"))
				printKV("session", *r.session)
				fmt.Println()
			default:
				*r.session = args[0]
				fmt.Println()
				fmt.Println(gcolor.HEX("#7fb88c").Sprint("  Session resumed"))
				printKV("session", *r.session)
				fmt.Println()
				state.SaveSession(*r.session)
			}
			return nil
		},
	}
}
