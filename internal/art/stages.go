// Package art provides ASCII art assets for the dojo CLI.
package art

import (
	"strings"
)

// ─── Growth Stage Definitions ────────────────────────────────────────────────
//
// 8 growth stages map 1:1 to belt ranks:
//   0 = White  (Seedling)
//   1 = Yellow (Sprout)
//   2 = Orange (Sapling)
//   3 = Green  (Young Tree)
//   4 = Blue   (Mature)
//   5 = Purple (Ancient)
//   6 = Brown  (Venerable)
//   7 = Black  (Eternal)
//
// For stages 0-2, sway is a no-op (all 3 phases return identical art).
// For stages 3-7, phase 0=left, 1=center, 2=right shifts the canopy ±1 col.

// clampStage clamps the stage to [0, 7].
func clampStage(s int) int {
	if s < 0 {
		return 0
	}
	if s > 7 {
		return 7
	}
	return s
}

// ─── Stage raw line tables ───────────────────────────────────────────────────
// Each entry is [3][]string — index by swayPhase (0=left, 1=center, 2=right).
// For stages 0-2 all three slices are identical.

// stage0Lines — Seedling (White Belt), 3 lines
var stage0Lines = [3][]string{
	{
		"    .",
		"    |",
		"   ___",
	},
	{
		"    .",
		"    |",
		"   ___",
	},
	{
		"    .",
		"    |",
		"   ___",
	},
}

// stage1Lines — Sprout (Yellow Belt), 4 lines
var stage1Lines = [3][]string{
	{
		"   .:.  ",
		"   '|'  ",
		"    |   ",
		"   [_]  ",
	},
	{
		"   .:.  ",
		"   '|'  ",
		"    |   ",
		"   [_]  ",
	},
	{
		"   .:.  ",
		"   '|'  ",
		"    |   ",
		"   [_]  ",
	},
}

// stage2Lines — Sapling (Orange Belt), 5 lines
var stage2Lines = [3][]string{
	{
		"  .:*:.  ",
		"  '.|.'  ",
		"    |    ",
		"    |    ",
		"   [_]   ",
	},
	{
		"  .:*:.  ",
		"  '.|.'  ",
		"    |    ",
		"    |    ",
		"   [_]   ",
	},
	{
		"  .:*:.  ",
		"  '.|.'  ",
		"    |    ",
		"    |    ",
		"   [_]   ",
	},
}

// stage3Lines — Young Tree (Green Belt), 7 lines
// Canopy is lines 0-2; trunk/pot are lines 3-6. Sway shifts lines 0-2.
var stage3Lines = [3][]string{
	// sway left
	{
		"   .:*~.   ",
		" .:*'~'*:. ",
		" '*:.,.:*' ",
		"     ||    ",
		"    /||\\  ",
		"  _[____]_ ",
		"           ",
	},
	// sway center (canonical)
	{
		"    .:*~.  ",
		"  .:*'~'*:.",
		"  '*:.,.:*'",
		"      ||   ",
		"     /||\\  ",
		"   _[____]_",
		"           ",
	},
	// sway right
	{
		"     .:*~. ",
		"   .:*'~'*:",
		"   '*:.,.:*",
		"      ||   ",
		"     /||\\  ",
		"   _[____]_",
		"           ",
	},
}

// stage4Lines — Mature (Blue Belt), 6 lines
// Matches SmallBonsai shape. Canopy: lines 0-2; trunk/pot: lines 3-5.
var stage4Lines = [3][]string{
	// sway left
	{
		"      .:*~*:.   ",
		"    .:*'~'*'~:. ",
		"   '*:..,~.,.:*'",
		"       |||      ",
		"       |||      ",
		"     _[___]_    ",
	},
	// sway center (canonical)
	{
		"       .:*~*:.  ",
		"     .:*'~'*'~:.",
		"    '*:..,~.,.:*'",
		"        |||     ",
		"        |||     ",
		"      _[___]_   ",
	},
	// sway right
	{
		"        .:*~*:. ",
		"      .:*'~'*'~:",
		"     '*:..,~.,.:*",
		"        |||     ",
		"        |||     ",
		"      _[___]_   ",
	},
}

// stage5Lines — Ancient (Purple Belt), 10 lines
// Matches MediumBonsai shape. Canopy: lines 0-4; trunk/pot: lines 5-9.
var stage5Lines = [3][]string{
	// sway left
	{
		"         .:*~.   ",
		"      .:*'~'*'~*:",
		"    .:*' ~.,.'~ '*",
		"   '*:.  .~*~.  .:*'",
		"     '~.,.,~.,  ,~'",
		"         /|\\    ",
		"         |||    ",
		"        /|||\\   ",
		"      _|=====|_ ",
		"       \\_____/  ",
	},
	// sway center (canonical)
	{
		"          .:*~.  ",
		"       .:*'~'*'~*:.",
		"     .:*' ~.,.'~ '*:.",
		"    '*:.  .~*~.  .:*'",
		"      '~.,.,~.,.,~'  ",
		"          /|\\   ",
		"          |||   ",
		"         /|||\\  ",
		"       _|=====|_",
		"        \\_____/ ",
	},
	// sway right
	{
		"           .:*~.",
		"        .:*'~'*'~*:.",
		"      .:*' ~.,.'~ '*:.",
		"     '*:.  .~*~.  .:*'",
		"       '~.,.,~.,.,~'  ",
		"          /|\\   ",
		"          |||   ",
		"         /|||\\  ",
		"       _|=====|_",
		"        \\_____/ ",
	},
}

// stage6Lines — Venerable (Brown Belt), 12 lines
// Bridges MediumBonsai and LargeBonsai. Canopy: lines 0-6; trunk/pot: lines 7-11.
var stage6Lines = [3][]string{
	// sway left
	{
		"            .:*~.    ",
		"         .:*'~*'~*:. ",
		"       .:*' ~.,.'~ '*:.",
		"     .:*' '~.*.'~.*  '*:",
		"    '*:.  .,~'*'~,.  .:*'",
		"      '~.,.,~*'*~,.,~'   ",
		"     '~.,.,~.,~.,~.,~'   ",
		"          __/|\\__  ",
		"            /|\\   ",
		"           /|||\\  ",
		"        _|=======|_",
		"         \\_______/ ",
	},
	// sway center (canonical)
	{
		"             .:*~.   ",
		"          .:*'~*'~*:.",
		"        .:*' ~.,.'~ '*:.",
		"      .:*' '~.*.'~.*  '*:.",
		"     '*:.  .,~'*'~,.  .:*'",
		"       '~.,.,~*'*~,.,~'   ",
		"      '~.,.,~.,~.,~.,~'   ",
		"           __/|\\__  ",
		"             /|\\   ",
		"            /|||\\  ",
		"         _|=======|_",
		"          \\_______/ ",
	},
	// sway right
	{
		"              .:*~.  ",
		"           .:*'~*'~*:.",
		"         .:*' ~.,.'~ '*:.",
		"       .:*' '~.*.'~.*  '*:.",
		"      '*:.  .,~'*'~,.  .:*'",
		"        '~.,.,~*'*~,.,~'   ",
		"       '~.,.,~.,~.,~.,~'   ",
		"           __/|\\__  ",
		"             /|\\   ",
		"            /|||\\  ",
		"         _|=======|_",
		"          \\_______/ ",
	},
}

// stage7Lines — Eternal (Black Belt), 14 lines
// Matches LargeBonsai shape. Canopy: lines 0-6; trunk/pot: lines 7-13.
var stage7Lines = [3][]string{
	// sway left
	{
		"             .:*~.     ",
		"          .:*'~*'*~'*:.",
		"       .:*' ~.,.'~.,  '*:.",
		"     .:*'  '~.*.'~.*  '*:.",
		"    '*:.  .,~'*'~,.   .:*' ",
		"   .:*'.,  ~.,.'~  ,.  '*:.",
		"     '~.,.,~*'*~.,.,~'  ",
		"         __/|\\__  ",
		"           /|\\   ",
		"           |||   ",
		"          /|||\\  ",
		"       __|=======|__",
		"       |  ~.,.,~  | ",
		"        \\_________/ ",
	},
	// sway center (canonical)
	{
		"              .:*~.    ",
		"           .:*'~*'*~'*:.",
		"        .:*' ~.,.'~.,  '*:.",
		"      .:*'  '~.*.'~.*  '*:.",
		"     '*:.  .,~'*'~,.   .:*'",
		"    .:*'.,  ~.,.'~  ,.  '*:.",
		"      '~.,.,~*'*~.,.,~'    ",
		"          __/|\\__   ",
		"            /|\\    ",
		"            |||    ",
		"           /|||\\   ",
		"        __|=======|__",
		"        |  ~.,.,~  | ",
		"         \\_________/ ",
	},
	// sway right
	{
		"               .:*~.   ",
		"            .:*'~*'*~'*:.",
		"         .:*' ~.,.'~.,  '*:.",
		"       .:*'  '~.*.'~.*  '*:.",
		"      '*:.  .,~'*'~,.   .:*'",
		"     .:*'.,  ~.,.'~  ,.  '*:.",
		"       '~.,.,~*'*~.,.,~'    ",
		"          __/|\\__   ",
		"            /|\\    ",
		"            |||    ",
		"           /|||\\   ",
		"        __|=======|__",
		"        |  ~.,.,~  | ",
		"         \\_________/ ",
	},
}

// allStages indexes stage → sway → lines for uniform access.
var allStages = [8][3][]string{
	stage0Lines,
	stage1Lines,
	stage2Lines,
	stage3Lines,
	stage4Lines,
	stage5Lines,
	stage6Lines,
	stage7Lines,
}

// ─── Public API ─────────────────────────────────────────────────────────────

// StageLines returns the raw uncolored lines for a growth stage (0-7).
// swayPhase: 0=left, 1=center, 2=right.
// Both arguments are clamped to valid ranges.
func StageLines(stage, swayPhase int) []string {
	stage = clampStage(stage)
	if swayPhase < 0 {
		swayPhase = 0
	}
	if swayPhase > 2 {
		swayPhase = 2
	}
	return allStages[stage][swayPhase]
}

// StageColored returns the colorized (lipgloss-styled) string for a growth stage.
// swayPhase: 0=left, 1=center, 2=right.
func StageColored(stage, swayPhase int) string {
	stage = clampStage(stage)
	lines := StageLines(stage, swayPhase)
	switch stage {
	case 0:
		return colorStage0(lines)
	case 1:
		return colorStage1(lines)
	case 2:
		return colorStage2(lines)
	case 3:
		return colorStage3(lines)
	case 4:
		return colorStage4(lines)
	case 5:
		return colorStage5(lines)
	case 6:
		return colorStage6(lines)
	case 7:
		return colorStage7(lines)
	}
	return strings.Join(lines, "\n")
}

// StageSpawnPoints returns canopy-edge (col, row) coordinates for petal spawning.
// Coordinates are (col, row) within the stage's own bounding box.
func StageSpawnPoints(stage int) [][2]int {
	stage = clampStage(stage)
	switch stage {
	case 0:
		// Just the seedling tip
		return [][2]int{{4, 0}}
	case 1:
		// Sprout crown — 3 points
		return [][2]int{{3, 0}, {4, 0}, {5, 0}}
	case 2:
		// Sapling crown spread
		return [][2]int{{2, 0}, {4, 0}, {6, 0}, {3, 1}, {5, 1}}
	case 3:
		// Young tree — canopy top and sides (lines 0-2)
		return [][2]int{
			{4, 0}, {6, 0}, {8, 0},
			{2, 1}, {4, 1}, {8, 1}, {10, 1},
			{1, 2}, {3, 2}, {9, 2}, {11, 2},
		}
	case 4:
		// Mature — canopy top and sides (lines 0-2)
		return [][2]int{
			{6, 0}, {9, 0}, {12, 0},
			{4, 1}, {7, 1}, {11, 1}, {14, 1},
			{3, 2}, {5, 2}, {12, 2}, {15, 2},
		}
	case 5:
		// Ancient — canopy lines 0-4
		return [][2]int{
			{10, 0}, {13, 0},
			{7, 1}, {10, 1}, {14, 1}, {17, 1},
			{4, 2}, {7, 2}, {15, 2}, {18, 2},
			{3, 3}, {6, 3}, {15, 3}, {19, 3},
			{2, 4}, {5, 4}, {14, 4}, {17, 4},
		}
	case 6:
		// Venerable — canopy lines 0-6
		return [][2]int{
			{12, 0}, {16, 0},
			{9, 1}, {13, 1}, {17, 1},
			{6, 2}, {9, 2}, {17, 2}, {21, 2},
			{4, 3}, {7, 3}, {17, 3}, {22, 3},
			{3, 4}, {6, 4}, {16, 4}, {22, 4},
			{2, 5}, {5, 5}, {17, 5}, {22, 5},
			{1, 6}, {4, 6}, {18, 6}, {22, 6},
		}
	case 7:
		// Eternal — canopy lines 0-6, densest
		return [][2]int{
			{14, 0}, {18, 0},
			{11, 1}, {15, 1}, {19, 1}, {23, 1},
			{8, 2}, {12, 2}, {20, 2}, {25, 2},
			{6, 3}, {10, 3}, {19, 3}, {25, 3},
			{4, 4}, {8, 4}, {18, 4}, {25, 4},
			{3, 5}, {7, 5}, {19, 5}, {27, 5},
			{2, 6}, {6, 6}, {20, 6}, {27, 6},
		}
	}
	return [][2]int{{0, 0}}
}

// StageWidth returns the maximum character width of a stage's lines.
func StageWidth(stage int) int {
	stage = clampStage(stage)
	lines := allStages[stage][1] // use center phase for canonical width
	max := 0
	for _, l := range lines {
		if len(l) > max {
			max = len(l)
		}
	}
	return max
}

// StageHeight returns the number of lines for a stage.
func StageHeight(stage int) int {
	stage = clampStage(stage)
	return len(allStages[stage][1])
}

// ─── Colorizers ─────────────────────────────────────────────────────────────
// Each colorizer receives the already-selected line slice and produces a
// newline-joined styled string using the package palette from bonsai.go.

func colorStage0(lines []string) string {
	var b strings.Builder
	// line 0: seedling bud tip
	b.WriteString(leafGold.Render(lines[0]))
	b.WriteByte('\n')
	// line 1: tiny stem
	b.WriteString(trunkBrown.Render(lines[1]))
	b.WriteByte('\n')
	// line 2: soil mound
	b.WriteString(soilColor.Render(lines[2]))
	return b.String()
}

func colorStage1(lines []string) string {
	var b strings.Builder
	b.WriteString(leafGold.Render(lines[0]))
	b.WriteByte('\n')
	b.WriteString(leafAmber.Render(lines[1]))
	b.WriteByte('\n')
	b.WriteString(trunkBrown.Render(lines[2]))
	b.WriteByte('\n')
	b.WriteString(potColor.Render(lines[3]))
	return b.String()
}

func colorStage2(lines []string) string {
	var b strings.Builder
	b.WriteString(leafGold.Render(lines[0]))
	b.WriteByte('\n')
	b.WriteString(leafAmber.Render(lines[1]))
	b.WriteByte('\n')
	b.WriteString(trunkBrown.Render(lines[2]))
	b.WriteByte('\n')
	b.WriteString(trunkDark.Render(lines[3]))
	b.WriteByte('\n')
	b.WriteString(potColor.Render(lines[4]))
	return b.String()
}

func colorStage3(lines []string) string {
	var b strings.Builder
	// canopy top
	b.WriteString(leafGold.Render(lines[0]))
	b.WriteByte('\n')
	// canopy mid
	b.WriteString(leafAmber.Render(lines[1]))
	b.WriteByte('\n')
	// canopy base
	b.WriteString(leafTerra.Render(lines[2]))
	b.WriteByte('\n')
	// trunk double-bar
	b.WriteString(trunkBrown.Render(lines[3]))
	b.WriteByte('\n')
	// branch spread
	b.WriteString(trunkBrown.Render(lines[4]))
	b.WriteByte('\n')
	// pot
	b.WriteString(potColor.Render(lines[5]))
	b.WriteByte('\n')
	// empty spacer if present
	if len(lines) > 6 {
		b.WriteString(lines[6])
	}
	return b.String()
}

func colorStage4(lines []string) string {
	// Mirrors SmallBonsai colorization
	var b strings.Builder
	b.WriteString("       ")
	b.WriteString(leafGold.Render(".:*~*:."))
	b.WriteByte('\n')
	b.WriteString("     ")
	b.WriteString(leafGold.Render(".:"))
	b.WriteString(leafAmber.Render("*'~'*'~"))
	b.WriteString(leafGold.Render(":."))
	b.WriteByte('\n')
	b.WriteString("    ")
	b.WriteString(leafAmber.Render("'*:."))
	b.WriteString(leafTerra.Render(".,~.,"))
	b.WriteString(leafAmber.Render(".:*'"))
	b.WriteByte('\n')
	b.WriteString("        ")
	b.WriteString(trunkBrown.Render("|||"))
	b.WriteByte('\n')
	b.WriteString("        ")
	b.WriteString(trunkDark.Render("|||"))
	b.WriteByte('\n')
	b.WriteString("      ")
	b.WriteString(potColor.Render("_[___]_"))
	return b.String()
}

func colorStage5(lines []string) string {
	// Mirrors MediumBonsai colorization
	var b strings.Builder
	b.WriteString("          ")
	b.WriteString(leafGold.Render(".:*~."))
	b.WriteByte('\n')
	b.WriteString("       ")
	b.WriteString(leafGold.Render(".:*'"))
	b.WriteString(leafAmber.Render("~'*'~"))
	b.WriteString(leafGold.Render("*:."))
	b.WriteByte('\n')
	b.WriteString("     ")
	b.WriteString(leafAmber.Render(".:*'"))
	b.WriteString(leafTerra.Render(" ~.,.'~ "))
	b.WriteString(leafAmber.Render("'*:."))
	b.WriteByte('\n')
	b.WriteString("    ")
	b.WriteString(leafTerra.Render("'*:."))
	b.WriteString(leafAmber.Render("  .~*~.  "))
	b.WriteString(leafTerra.Render(".:*'"))
	b.WriteByte('\n')
	b.WriteString("      ")
	b.WriteString(leafTerra.Render("'~.,"))
	b.WriteString(leafAmber.Render(".,~.,"))
	b.WriteString(leafTerra.Render(",~'"))
	b.WriteByte('\n')
	b.WriteString("          ")
	b.WriteString(trunkBrown.Render("/|\\"))
	b.WriteByte('\n')
	b.WriteString("          ")
	b.WriteString(trunkBrown.Render("|||"))
	b.WriteByte('\n')
	b.WriteString("         ")
	b.WriteString(trunkDark.Render("/|||\\"))
	b.WriteByte('\n')
	b.WriteString("       ")
	b.WriteString(potColor.Render("_|=====|_"))
	b.WriteByte('\n')
	b.WriteString("        ")
	b.WriteString(potColor.Render("\\_____/"))
	return b.String()
}

func colorStage6(lines []string) string {
	// Venerable — intermediate between Medium and Large
	var b strings.Builder
	// line 0: crown tip
	b.WriteString("             ")
	b.WriteString(leafGold.Render(".:*~."))
	b.WriteByte('\n')
	// line 1: upper canopy
	b.WriteString("          ")
	b.WriteString(leafGold.Render(".:*'~*'~*:."))
	b.WriteByte('\n')
	// line 2: upper-mid canopy
	b.WriteString("        ")
	b.WriteString(leafGold.Render(".:*'"))
	b.WriteString(leafAmber.Render(" ~.,.'~ "))
	b.WriteString(leafGold.Render("'*:."))
	b.WriteByte('\n')
	// line 3: mid canopy
	b.WriteString("      ")
	b.WriteString(leafAmber.Render(".:*'"))
	b.WriteString(leafTerra.Render(" '~.*.'~.* "))
	b.WriteString(leafAmber.Render("'*:."))
	b.WriteByte('\n')
	// line 4: lower canopy
	b.WriteString("     ")
	b.WriteString(leafTerra.Render("'*:."))
	b.WriteString(leafAmber.Render("  .,~'*'~,.  "))
	b.WriteString(leafTerra.Render(".:*'"))
	b.WriteByte('\n')
	// line 5: canopy floor
	b.WriteString("       ")
	b.WriteString(leafTerra.Render("'~.,"))
	b.WriteString(leafAmber.Render(".,~*'*~,."))
	b.WriteString(leafTerra.Render(",~'"))
	b.WriteByte('\n')
	// line 6: canopy base fringe
	b.WriteString("      ")
	b.WriteString(leafTerra.Render("'~.,.,~.,~.,~.,~'"))
	b.WriteByte('\n')
	// line 7: branch spread
	b.WriteString("          ")
	b.WriteString(trunkBrown.Render("__/|\\__"))
	b.WriteByte('\n')
	// line 8: upper trunk
	b.WriteString("            ")
	b.WriteString(trunkBrown.Render("/|\\"))
	b.WriteByte('\n')
	// line 9: lower trunk
	b.WriteString("           ")
	b.WriteString(trunkDark.Render("/|||\\"))
	b.WriteByte('\n')
	// line 10: pot rim
	b.WriteString("         ")
	b.WriteString(potColor.Render("_|=======|_"))
	b.WriteByte('\n')
	// line 11: pot base
	b.WriteString("          ")
	b.WriteString(potColor.Render("\\_______/"))
	return b.String()
}

func colorStage7(lines []string) string {
	// Mirrors LargeBonsai colorization
	var b strings.Builder
	b.WriteString("              ")
	b.WriteString(leafGold.Render(".:*~."))
	b.WriteByte('\n')
	b.WriteString("           ")
	b.WriteString(leafGold.Render(".:*'~"))
	b.WriteString(leafAmber.Render("*'*"))
	b.WriteString(leafGold.Render("~'*:."))
	b.WriteByte('\n')
	b.WriteString("        ")
	b.WriteString(leafGold.Render(".:*'"))
	b.WriteString(leafAmber.Render(" ~.,.'~.,"))
	b.WriteString(leafGold.Render(" '*:."))
	b.WriteByte('\n')
	b.WriteString("      ")
	b.WriteString(leafAmber.Render(".:*'"))
	b.WriteString(leafTerra.Render("  '~.*.'~.*  "))
	b.WriteString(leafAmber.Render("'*:."))
	b.WriteByte('\n')
	b.WriteString("     ")
	b.WriteString(leafAmber.Render("'*:."))
	b.WriteString(leafTerra.Render("  .,~'*'~,.   "))
	b.WriteString(leafAmber.Render(".:*'"))
	b.WriteByte('\n')
	b.WriteString("    ")
	b.WriteString(leafTerra.Render(".:*'"))
	b.WriteString(leafAmber.Render(".,"))
	b.WriteString(leafTerra.Render("  ~.,.'~  "))
	b.WriteString(leafAmber.Render(",."))
	b.WriteString(leafTerra.Render("'*:."))
	b.WriteByte('\n')
	b.WriteString("      ")
	b.WriteString(leafTerra.Render("'~.,"))
	b.WriteString(leafAmber.Render(".,~*'*~,."))
	b.WriteString(leafTerra.Render(",~'"))
	b.WriteByte('\n')
	b.WriteString("          ")
	b.WriteString(trunkBrown.Render("__/|\\__"))
	b.WriteByte('\n')
	b.WriteString("            ")
	b.WriteString(trunkBrown.Render("/|\\"))
	b.WriteByte('\n')
	b.WriteString("            ")
	b.WriteString(trunkBrown.Render("|||"))
	b.WriteByte('\n')
	b.WriteString("           ")
	b.WriteString(trunkDark.Render("/|||\\"))
	b.WriteByte('\n')
	b.WriteString("        ")
	b.WriteString(potColor.Render("__|=======|__"))
	b.WriteByte('\n')
	b.WriteString("        ")
	b.WriteString(potColor.Render("|"))
	b.WriteString(mossColor.Render("  "))
	b.WriteString(soilColor.Render("~.,.,~"))
	b.WriteString(mossColor.Render("  "))
	b.WriteString(potColor.Render("|"))
	b.WriteByte('\n')
	b.WriteString("         ")
	b.WriteString(potColor.Render("\\_________/"))
	return b.String()
}
