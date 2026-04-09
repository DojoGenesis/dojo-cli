package commands

// cmd_spirit.go — /sensei, /card commands and belt-up notification.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DojoGenesis/dojo-cli/internal/art"
	"github.com/DojoGenesis/dojo-cli/internal/spirit"
	"github.com/DojoGenesis/dojo-cli/internal/state"
	gcolor "github.com/gookit/color"
)

// ─── /sensei ────────────────────────────────────────────────────────────────

func (r *Registry) senseiCmd() Command {
	return Command{
		Name:    "sensei",
		Aliases: []string{"koan", "wisdom"},
		Usage:   "/sensei",
		Short:   "Receive a koan from the sensei",
		Run: func(ctx context.Context, args []string) error {
			st, err := state.Load()
			if err != nil {
				return fmt.Errorf("loading state: %w", err)
			}

			belt := spirit.CurrentBelt(st.Spirit.XP)
			koan := spirit.RandomKoan(belt.Rank, time.Now())
			unlocked := spirit.KoanCount(belt.Rank)
			total := spirit.TotalKoans()

			fmt.Println()
			fmt.Print(art.SmallBonsaiString())
			fmt.Println()
			fmt.Println(gcolor.HEX("#e8b04a").Sprint("  " + koan))
			fmt.Println()
			fmt.Println(gcolor.HEX("#94a3b8").Sprintf(
				"  \u2014 %s Belt (%d/%d koans unlocked)",
				belt.Name, unlocked, total,
			))
			fmt.Println()
			return nil
		},
	}
}

// ─── /card ──────────────────────────────────────────────────────────────────

func (r *Registry) cardCmd() Command {
	return Command{
		Name:    "card",
		Aliases: []string{"profile", "rank", "belt"},
		Usage:   "/card",
		Short:   "Show your dojo profile card",
		Run: func(ctx context.Context, args []string) error {
			st, err := state.Load()
			if err != nil {
				return fmt.Errorf("loading state: %w", err)
			}

			sp := st.Spirit
			belt := spirit.CurrentBelt(sp.XP)
			next, _ := spirit.NextBelt(sp.XP)
			pct := spirit.ProgressPercent(sp.XP)
			achievements := spirit.UnlockedAchievements(&sp)

			// XP display
			var xpLine string
			if next == nil {
				xpLine = fmt.Sprintf("%d / MAX", sp.XP)
			} else {
				xpLine = fmt.Sprintf("%d / %d", sp.XP, next.Threshold)
			}

			// Progress bar: 20 chars wide, Unicode blocks
			barWidth := 20
			filled := int(pct * float64(barWidth))
			if filled > barWidth {
				filled = barWidth
			}
			bar := strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", barWidth-filled)
			pctStr := fmt.Sprintf("%d%%", int(pct*100))

			// Member since
			memberSince := "unknown"
			if sp.MemberSince != "" {
				if t, err := time.Parse("2006-01-02", sp.MemberSince); err == nil {
					memberSince = t.Format("Jan 2006")
				} else if t, err := time.Parse(time.RFC3339, sp.MemberSince); err == nil {
					memberSince = t.Format("Jan 2006")
				}
			}

			// Belt name colored by belt color
			beltDisplay := gcolor.HEX(belt.Color).Sprintf("%s %s", belt.Name, belt.Title)

			w := 40 // inner width (between box edges)

			fmt.Println()
			fmt.Printf("  \u256d%s\u256e\n", strings.Repeat("\u2500", w))
			fmt.Printf("  \u2502  %-*s\u2502\n", w-2, "DOJO PROFILE CARD")
			fmt.Printf("  \u251c%s\u2524\n", strings.Repeat("\u2500", w))
			printCardRow(w, "Belt:", beltDisplay, len(beltDisplay)-visLen(beltDisplay))
			printCardRow(w, "XP:", xpLine, 0)
			printCardRow(w, "", fmt.Sprintf("[%s] %s", bar, pctStr), 0)
			printCardEmpty(w)
			printCardRow(w, "Streak:", fmt.Sprintf("%d days", sp.StreakDays), 0)
			printCardRow(w, "Sessions:", fmt.Sprintf("%d  Since: %s", sp.TotalSessions, memberSince), 0)
			printCardEmpty(w)
			printCardRow(w, "Achievements:", "", 0)

			if len(achievements) == 0 {
				printCardRow(w, "", "None yet \u2014 keep practicing!", 0)
			} else {
				// Show max 6, newest first, in rows of 2
				limit := len(achievements)
				if limit > 6 {
					limit = 6
				}
				for i := 0; i < limit; i += 2 {
					a1 := achievements[i]
					col1 := fmt.Sprintf("%s %s", a1.Icon, a1.Name)
					if i+1 < limit {
						a2 := achievements[i+1]
						col2 := fmt.Sprintf("%s %s", a2.Icon, a2.Name)
						printCardRow(w, "", fmt.Sprintf("%-18s %s", col1, col2), 0)
					} else {
						printCardRow(w, "", col1, 0)
					}
				}
			}

			fmt.Printf("  \u2570%s\u256f\n", strings.Repeat("\u2500", w))
			fmt.Println()
			return nil
		},
	}
}

// printCardRow prints a row inside the card frame.
// extraWidth accounts for ANSI escape sequences in the value that add length
// without adding visible characters.
func printCardRow(w int, label, value string, extraWidth int) {
	content := fmt.Sprintf("  %-11s%-*s", label, w-13+extraWidth, value)
	fmt.Printf("  \u2502%s\u2502\n", content)
}

// printCardEmpty prints an empty row inside the card frame.
func printCardEmpty(w int) {
	fmt.Printf("  \u2502%-*s\u2502\n", w, "")
}

// visLen returns the visible length of a string, stripping ANSI escape codes.
func visLen(s string) int {
	inEsc := false
	n := 0
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		n++
	}
	return n
}

// ─── Belt Promotion Notification ────────────────────────────────────────────

// NotifyBeltUp prints a belt promotion notification.
// Called from the REPL when a belt promotion occurs.
func NotifyBeltUp(belt spirit.Belt) {
	quote := spirit.BeltQuote(belt.Rank)
	beltDisplay := gcolor.HEX(belt.Color).Sprintf("%s %s", belt.Name, belt.Title)

	w := 39

	fmt.Println()
	fmt.Printf("  \u250c%s\u2510\n", strings.Repeat("\u2500", w))
	fmt.Printf("  \u2502  %-*s\u2502\n", w-2, "BELT PROMOTION")

	// Belt name row with ANSI-aware padding
	beltLine := fmt.Sprintf("You are now: %s", beltDisplay)
	extra := len(beltLine) - visLen(beltLine)
	fmt.Printf("  \u2502  %-*s\u2502\n", w-2+extra, beltLine)

	// Quote row
	quoteLine := fmt.Sprintf("\"%s\"", quote)
	if len(quoteLine) > w-4 {
		quoteLine = quoteLine[:w-5] + "\u2026\""
	}
	fmt.Printf("  \u2502  %-*s\u2502\n", w-2, quoteLine)

	fmt.Printf("  \u2514%s\u2518\n", strings.Repeat("\u2500", w))
	fmt.Println()
}
