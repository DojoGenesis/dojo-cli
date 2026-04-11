package commands

// cmd_memory.go — /trail and /snapshot commands and their helpers.

import (
	"context"
	"fmt"
	"strings"

	"github.com/DojoGenesis/cli/internal/client"
	gcolor "github.com/gookit/color"
)

// ─── /trail ─────────────────────────────────────────────────────────────────

func (r *Registry) trailCmd() Command {
	return Command{
		Name:  "trail",
		Usage: "/trail [add <text>|rm <id>|search <query>]",
		Short: "Show memory timeline or add/remove/search memories",
		Run: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				// default: list
				memories, err := r.gw.Memories(ctx)
				if err != nil {
					return fmt.Errorf("could not fetch memory trail: %w", err)
				}
				fmt.Println()
				gcolor.Bold.Print(gcolor.HEX("#e8b04a").Sprintf("  Memory Trail (%d)\n\n", len(memories)))
				if len(memories) == 0 {
					fmt.Println(gcolor.HEX("#94a3b8").Sprint("  No memory entries yet."))
					fmt.Println()
					return nil
				}
				for _, m := range memories {
					fmt.Printf("  %s  %s\n",
						gcolor.HEX("#94a3b8").Sprintf("%-20s", m.CreatedAt),
						gcolor.White.Sprint(truncate(m.Content, 80)),
					)
				}
				fmt.Println()
				return nil
			}

			sub := strings.ToLower(args[0])
			switch sub {
			case "add":
				// /trail add <text...>
				if len(args) < 2 {
					return fmt.Errorf("usage: /trail add <text>")
				}
				text := strings.Join(args[1:], " ")
				mem, err := r.gw.StoreMemory(ctx, client.StoreMemoryRequest{Content: text})
				if err != nil {
					return fmt.Errorf("could not store memory: %w", err)
				}
				fmt.Println()
				fmt.Println(gcolor.HEX("#7fb88c").Sprint("  Memory stored"))
				if mem != nil {
					printKV("id", mem.ID)
				}
				fmt.Println()

			case "rm":
				// /trail rm <id>
				if len(args) < 2 {
					return fmt.Errorf("usage: /trail rm <id>")
				}
				id := args[1]
				if err := r.gw.DeleteMemory(ctx, id); err != nil {
					return fmt.Errorf("could not delete memory: %w", err)
				}
				fmt.Println()
				fmt.Println(gcolor.HEX("#7fb88c").Sprint("  Memory deleted"))
				fmt.Println()

			case "search":
				// /trail search <query...>
				if len(args) < 2 {
					return fmt.Errorf("usage: /trail search <query>")
				}
				query := strings.Join(args[1:], " ")
				results, err := r.gw.SearchMemories(ctx, query)
				if err != nil {
					return fmt.Errorf("could not search memories: %w", err)
				}
				fmt.Println()
				gcolor.Bold.Print(gcolor.HEX("#e8b04a").Sprintf("  Search results (%d)\n\n", len(results)))
				if len(results) == 0 {
					fmt.Println(gcolor.HEX("#94a3b8").Sprint("  No results found."))
					fmt.Println()
					return nil
				}
				for _, m := range results {
					fmt.Printf("  %s  %s\n",
						gcolor.HEX("#94a3b8").Sprintf("%-24s", m.ID),
						gcolor.White.Sprint(truncate(m.Content, 80)),
					)
				}
				fmt.Println()

			default:
				return fmt.Errorf("unknown trail subcommand %q — use: add, rm, search", sub)
			}
			return nil
		},
	}
}

// ─── /snapshot ───────────────────────────────────────────────────────────────

func (r *Registry) snapshotCmd() Command {
	return Command{
		Name:  "snapshot",
		Usage: "/snapshot [save|restore <id>|export <id>|rm <id>]",
		Short: "List, save, restore, export, or delete memory snapshots",
		Run: func(ctx context.Context, args []string) error {
			sub := "ls"
			if len(args) > 0 {
				sub = strings.ToLower(args[0])
			}

			switch sub {
			case "save":
				snap, err := r.gw.CreateSnapshot(ctx, *r.session)
				if err != nil {
					return fmt.Errorf("could not create snapshot: %w", err)
				}
				fmt.Println()
				fmt.Println(gcolor.HEX("#7fb88c").Sprint("  Snapshot saved"))
				if snap != nil {
					printKV("id", snap.ID)
					printKV("session", snap.SessionID)
					printKV("created", snap.CreatedAt)
				}
				fmt.Println()

			case "restore":
				if len(args) < 2 {
					return fmt.Errorf("usage: /snapshot restore <id>")
				}
				id := args[1]
				if err := r.gw.RestoreSnapshot(ctx, id); err != nil {
					return fmt.Errorf("could not restore snapshot: %w", err)
				}
				fmt.Println()
				fmt.Println(gcolor.HEX("#7fb88c").Sprint("  Snapshot restored"))
				printKV("id", id)
				fmt.Println()

			case "export":
				if len(args) < 2 {
					return fmt.Errorf("usage: /snapshot export <id>")
				}
				id := args[1]
				data, err := r.gw.ExportSnapshot(ctx, id)
				if err != nil {
					return fmt.Errorf("could not export snapshot: %w", err)
				}
				fmt.Println(string(data))

			case "rm":
				if len(args) < 2 {
					return fmt.Errorf("usage: /snapshot rm <id>")
				}
				id := args[1]
				if err := r.gw.DeleteSnapshot(ctx, id); err != nil {
					return fmt.Errorf("could not delete snapshot: %w", err)
				}
				fmt.Println()
				fmt.Println(gcolor.HEX("#7fb88c").Sprint("  Snapshot deleted"))
				fmt.Println()

			default: // ls
				snaps, err := r.gw.ListSnapshots(ctx, *r.session)
				if err != nil {
					return fmt.Errorf("could not fetch snapshots: %w", err)
				}
				fmt.Println()
				gcolor.Bold.Print(gcolor.HEX("#e8b04a").Sprintf("  Snapshots (%d)\n\n", len(snaps)))
				if len(snaps) == 0 {
					fmt.Println(gcolor.HEX("#94a3b8").Sprint("  No snapshots found. Use /snapshot save to create one."))
					fmt.Println()
					return nil
				}
				for _, s := range snaps {
					fmt.Printf("  %s  %s\n",
						gcolor.HEX("#f4a261").Sprintf("%-36s", s.ID),
						gcolor.HEX("#94a3b8").Sprint(s.CreatedAt),
					)
				}
				fmt.Println()
			}
			return nil
		},
	}
}
