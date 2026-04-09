package spirit

import (
	"strings"
	"testing"
	"time"
)

// ─── Belt Lookup ─────────────────────────────────────────────────────────────

func TestCurrentBelt(t *testing.T) {
	tests := []struct {
		name     string
		xp       int
		wantRank int
		wantName string
	}{
		{"zero XP is White", 0, 0, "White"},
		{"500 XP is White", 500, 0, "White"},
		{"1000 XP is Yellow", 1000, 1, "Yellow"},
		{"2999 XP is Yellow", 2999, 1, "Yellow"},
		{"3000 XP is Orange", 3000, 2, "Orange"},
		{"6000 XP is Green", 6000, 3, "Green"},
		{"10000 XP is Blue", 10000, 4, "Blue"},
		{"15000 XP is Purple", 15000, 5, "Purple"},
		{"25000 XP is Brown", 25000, 6, "Brown"},
		{"50000 XP is Black", 50000, 7, "Black"},
		{"99999 XP is still Black", 99999, 7, "Black"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			belt := CurrentBelt(tt.xp)
			if belt.Rank != tt.wantRank {
				t.Errorf("CurrentBelt(%d).Rank = %d, want %d", tt.xp, belt.Rank, tt.wantRank)
			}
			if belt.Name != tt.wantName {
				t.Errorf("CurrentBelt(%d).Name = %q, want %q", tt.xp, belt.Name, tt.wantName)
			}
		})
	}
}

func TestCurrentBelt_Boundaries(t *testing.T) {
	// Exact threshold values must land ON the belt, not below it.
	for _, b := range Belts {
		got := CurrentBelt(b.Threshold)
		if got.Rank != b.Rank {
			t.Errorf("CurrentBelt(%d) = %q (rank %d), want %q (rank %d)",
				b.Threshold, got.Name, got.Rank, b.Name, b.Rank)
		}
	}

	// One below each threshold (except White at 0) should be the previous belt.
	for i := 1; i < len(Belts); i++ {
		xp := Belts[i].Threshold - 1
		got := CurrentBelt(xp)
		wantRank := Belts[i-1].Rank
		if got.Rank != wantRank {
			t.Errorf("CurrentBelt(%d) = rank %d, want rank %d (one below %s threshold)",
				xp, got.Rank, wantRank, Belts[i].Name)
		}
	}
}

// ─── Next Belt ───────────────────────────────────────────────────────────────

func TestNextBelt(t *testing.T) {
	t.Run("White belt next is Yellow", func(t *testing.T) {
		next, remaining := NextBelt(0)
		if next == nil {
			t.Fatal("NextBelt(0) returned nil, want Yellow")
		}
		if next.Name != "Yellow" {
			t.Errorf("NextBelt(0) name = %q, want Yellow", next.Name)
		}
		if remaining != 1000 {
			t.Errorf("NextBelt(0) remaining = %d, want 1000", remaining)
		}
	})

	t.Run("Mid-Yellow next is Orange", func(t *testing.T) {
		next, remaining := NextBelt(2000)
		if next == nil {
			t.Fatal("NextBelt(2000) returned nil, want Orange")
		}
		if next.Name != "Orange" {
			t.Errorf("NextBelt(2000) name = %q, want Orange", next.Name)
		}
		if remaining != 1000 {
			t.Errorf("NextBelt(2000) remaining = %d, want 1000", remaining)
		}
	})

	t.Run("Black belt returns nil", func(t *testing.T) {
		next, remaining := NextBelt(50000)
		if next != nil {
			t.Errorf("NextBelt(50000) = %v, want nil", next)
		}
		if remaining != 0 {
			t.Errorf("NextBelt(50000) remaining = %d, want 0", remaining)
		}
	})

	t.Run("Beyond Black returns nil", func(t *testing.T) {
		next, _ := NextBelt(100000)
		if next != nil {
			t.Errorf("NextBelt(100000) = %v, want nil", next)
		}
	})
}

// ─── Progress ────────────────────────────────────────────────────────────────

func TestProgressPercent(t *testing.T) {
	t.Run("start of White is 0.0", func(t *testing.T) {
		pct := ProgressPercent(0)
		if pct != 0.0 {
			t.Errorf("ProgressPercent(0) = %f, want 0.0", pct)
		}
	})

	t.Run("midpoint of White-to-Yellow", func(t *testing.T) {
		// White threshold=0, Yellow threshold=1000, midpoint=500 -> 0.5
		pct := ProgressPercent(500)
		if pct < 0.49 || pct > 0.51 {
			t.Errorf("ProgressPercent(500) = %f, want ~0.5", pct)
		}
	})

	t.Run("Black belt is 1.0", func(t *testing.T) {
		pct := ProgressPercent(50000)
		if pct != 1.0 {
			t.Errorf("ProgressPercent(50000) = %f, want 1.0", pct)
		}
	})

	t.Run("beyond Black is 1.0", func(t *testing.T) {
		pct := ProgressPercent(80000)
		if pct != 1.0 {
			t.Errorf("ProgressPercent(80000) = %f, want 1.0", pct)
		}
	})
}

func TestProgressBar(t *testing.T) {
	t.Run("0% filled", func(t *testing.T) {
		bar := ProgressBar(0, 20)
		if len(bar) != 20 {
			t.Fatalf("ProgressBar width = %d, want 20", len(bar))
		}
		if strings.Count(bar, "#") != 0 {
			t.Errorf("ProgressBar(0,20) has %d filled, want 0", strings.Count(bar, "#"))
		}
		if strings.Count(bar, "-") != 20 {
			t.Errorf("ProgressBar(0,20) has %d empty, want 20", strings.Count(bar, "-"))
		}
	})

	t.Run("~50% filled", func(t *testing.T) {
		// 500 XP = 50% of White-to-Yellow
		bar := ProgressBar(500, 20)
		filled := strings.Count(bar, "#")
		if filled < 9 || filled > 11 {
			t.Errorf("ProgressBar(500,20) filled = %d, want ~10", filled)
		}
	})

	t.Run("100% filled at Black", func(t *testing.T) {
		bar := ProgressBar(50000, 20)
		if strings.Count(bar, "#") != 20 {
			t.Errorf("ProgressBar(50000,20) filled = %d, want 20", strings.Count(bar, "#"))
		}
	})

	t.Run("bar length matches width", func(t *testing.T) {
		for _, w := range []int{5, 10, 30, 50} {
			bar := ProgressBar(1500, w)
			if len(bar) != w {
				t.Errorf("ProgressBar width=%d, got len %d", w, len(bar))
			}
		}
	})
}

// ─── XP Actions ──────────────────────────────────────────────────────────────

func TestXPForAction(t *testing.T) {
	known := map[string]int{
		"session_start":      25,
		"command_run":        5,
		"chat_message":       5,
		"agent_dispatched":   50,
		"skill_invoked":      30,
		"artifact_saved":     20,
		"project_created":    100,
		"practice_completed": 40,
		"seed_planted":       15,
		"plugin_installed":   75,
		"model_changed":      10,
		"phase_advanced":     60,
		"belt_promotion":     200,
	}
	for action, want := range known {
		t.Run(action, func(t *testing.T) {
			got := XPForAction(action)
			if got != want {
				t.Errorf("XPForAction(%q) = %d, want %d", action, got, want)
			}
		})
	}

	t.Run("unknown action returns 0", func(t *testing.T) {
		got := XPForAction("nonexistent_action")
		if got != 0 {
			t.Errorf("XPForAction(unknown) = %d, want 0", got)
		}
	})
}

// ─── Award XP ────────────────────────────────────────────────────────────────

func TestAwardXP_NoBeltUp(t *testing.T) {
	s := &SpiritState{XP: 100}
	beltedUp, belt := AwardXP(s, 50)
	if beltedUp {
		t.Error("AwardXP: expected no belt promotion for small award")
	}
	if s.XP != 150 {
		t.Errorf("XP = %d, want 150", s.XP)
	}
	if belt.Rank != 0 {
		t.Errorf("belt rank = %d, want 0 (White)", belt.Rank)
	}
}

func TestAwardXP_BeltUp(t *testing.T) {
	s := &SpiritState{XP: 999}
	beltedUp, belt := AwardXP(s, 2)
	if !beltedUp {
		t.Error("AwardXP: expected belt promotion from 999+2=1001")
	}
	if belt.Rank != 1 {
		t.Errorf("belt rank = %d, want 1 (Yellow)", belt.Rank)
	}
	if s.XP != 1001 {
		t.Errorf("XP = %d, want 1001", s.XP)
	}
}

func TestAwardXP_MultipleBeltUp(t *testing.T) {
	// Jump from White (0) past Yellow (1000) and Orange (3000) to land in Green (6000+)
	s := &SpiritState{XP: 0}
	beltedUp, belt := AwardXP(s, 7000)
	if !beltedUp {
		t.Error("AwardXP: expected belt promotion on large XP award")
	}
	if belt.Rank != 3 {
		t.Errorf("belt rank = %d, want 3 (Green)", belt.Rank)
	}
	if belt.Name != "Green" {
		t.Errorf("belt name = %q, want Green", belt.Name)
	}
}

func TestAwardXP_Zero(t *testing.T) {
	s := &SpiritState{XP: 500}

	t.Run("zero amount", func(t *testing.T) {
		beltedUp, _ := AwardXP(s, 0)
		if beltedUp {
			t.Error("AwardXP(0): should not promote")
		}
		if s.XP != 500 {
			t.Errorf("XP = %d, want 500 (unchanged)", s.XP)
		}
	})

	t.Run("negative amount", func(t *testing.T) {
		beltedUp, _ := AwardXP(s, -10)
		if beltedUp {
			t.Error("AwardXP(-10): should not promote")
		}
		if s.XP != 500 {
			t.Errorf("XP = %d, want 500 (unchanged)", s.XP)
		}
	})
}

// ─── Streak System ───────────────────────────────────────────────────────────

func fixedTime(dateStr string) time.Time {
	t, _ := time.Parse("2006-01-02", dateStr)
	return t
}

func TestUpdateStreak_FirstSession(t *testing.T) {
	s := &SpiritState{}
	now := fixedTime("2026-04-09")
	bonus := UpdateStreak(s, now)

	if bonus != 0 {
		t.Errorf("first session bonus = %d, want 0", bonus)
	}
	if s.StreakDays != 1 {
		t.Errorf("streak = %d, want 1", s.StreakDays)
	}
	if s.LastActiveDate != "2026-04-09" {
		t.Errorf("LastActiveDate = %q, want 2026-04-09", s.LastActiveDate)
	}
}

func TestUpdateStreak_ConsecutiveDay(t *testing.T) {
	s := &SpiritState{
		StreakDays:     3,
		LastActiveDate: "2026-04-08",
	}
	now := fixedTime("2026-04-09")
	bonus := UpdateStreak(s, now)

	if s.StreakDays != 4 {
		t.Errorf("streak = %d, want 4", s.StreakDays)
	}
	// bonus = 4 * 5 = 20
	if bonus != 20 {
		t.Errorf("bonus = %d, want 20", bonus)
	}
}

func TestUpdateStreak_GraceWindow(t *testing.T) {
	// Last active 2 days ago -- 48h grace window should keep streak alive.
	s := &SpiritState{
		StreakDays:     5,
		LastActiveDate: "2026-04-07",
	}
	now := fixedTime("2026-04-09")
	bonus := UpdateStreak(s, now)

	if s.StreakDays != 6 {
		t.Errorf("streak = %d, want 6 (grace window)", s.StreakDays)
	}
	// bonus = 6 * 5 = 30
	if bonus != 30 {
		t.Errorf("bonus = %d, want 30", bonus)
	}
}

func TestUpdateStreak_Broken(t *testing.T) {
	// Last active 3+ days ago -- streak should reset.
	s := &SpiritState{
		StreakDays:     10,
		LastActiveDate: "2026-04-05",
	}
	now := fixedTime("2026-04-09")
	bonus := UpdateStreak(s, now)

	if s.StreakDays != 1 {
		t.Errorf("streak = %d, want 1 (reset)", s.StreakDays)
	}
	// bonus = 1 * 5 = 5
	if bonus != 5 {
		t.Errorf("bonus = %d, want 5", bonus)
	}
}

func TestUpdateStreak_SameDay(t *testing.T) {
	now := fixedTime("2026-04-09")
	s := &SpiritState{
		StreakDays:     3,
		LastActiveDate: "2026-04-08",
	}

	// First call today: should get a bonus.
	bonus1 := UpdateStreak(s, now)
	if bonus1 == 0 {
		t.Error("first call today should return bonus > 0")
	}

	// Second call same day: StreakBonusDate already set, should return 0.
	bonus2 := UpdateStreak(s, now)
	if bonus2 != 0 {
		t.Errorf("second call same day bonus = %d, want 0", bonus2)
	}
}

// ─── Achievements ────────────────────────────────────────────────────────────

func TestCheckAchievements_FirstSteps(t *testing.T) {
	s := &SpiritState{TotalSessions: 1}
	now := time.Now()
	newly := CheckAchievements(s, now)

	found := false
	for _, a := range newly {
		if a.ID == "first_steps" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'first_steps' achievement to be unlocked")
	}
	if _, ok := s.Unlocked["first_steps"]; !ok {
		t.Error("first_steps not recorded in Unlocked map")
	}
}

func TestCheckAchievements_AlreadyUnlocked(t *testing.T) {
	s := &SpiritState{
		TotalSessions: 1,
		Unlocked:      map[string]string{"first_steps": "2026-01-01T00:00:00Z"},
	}
	now := time.Now()
	newly := CheckAchievements(s, now)

	for _, a := range newly {
		if a.ID == "first_steps" {
			t.Error("first_steps should NOT be returned again when already unlocked")
		}
	}
}

func TestCheckAchievements_Multiple(t *testing.T) {
	s := &SpiritState{
		TotalSessions: 1,
		TotalSeeds:    10,
		TotalCommands: 100,
	}
	now := time.Now()
	newly := CheckAchievements(s, now)

	ids := make(map[string]bool)
	for _, a := range newly {
		ids[a.ID] = true
	}

	for _, want := range []string{"first_steps", "green_thumb", "commander"} {
		if !ids[want] {
			t.Errorf("expected %q achievement to be unlocked", want)
		}
	}
}

// ─── Koans ───────────────────────────────────────────────────────────────────

func TestRandomKoan(t *testing.T) {
	now := time.Now()

	t.Run("non-empty result", func(t *testing.T) {
		for rank := 0; rank <= 7; rank++ {
			k := RandomKoan(rank, now)
			if k == "" {
				t.Errorf("RandomKoan(rank=%d) returned empty string", rank)
			}
		}
	})

	t.Run("White belt only gets rank 0 koans", func(t *testing.T) {
		whiteKoans := make(map[string]bool)
		for _, k := range koans[0] {
			whiteKoans[k] = true
		}
		// Call many times with different seeds to check we never get a non-rank-0 koan.
		for i := 0; i < 200; i++ {
			ts := now.Add(time.Duration(i) * time.Millisecond)
			k := RandomKoan(0, ts)
			if !whiteKoans[k] {
				t.Errorf("RandomKoan(0) returned non-White koan: %q", k)
				break
			}
		}
	})
}

func TestKoanCount(t *testing.T) {
	t.Run("White belt count", func(t *testing.T) {
		got := KoanCount(0)
		want := len(koans[0])
		if got != want {
			t.Errorf("KoanCount(0) = %d, want %d", got, want)
		}
		if got != 5 {
			t.Errorf("KoanCount(0) = %d, want 5", got)
		}
	})

	t.Run("Black belt gets all koans", func(t *testing.T) {
		got := KoanCount(7)
		total := TotalKoans()
		if got != total {
			t.Errorf("KoanCount(7) = %d, want TotalKoans() = %d", got, total)
		}
	})

	t.Run("count increases with rank", func(t *testing.T) {
		prev := KoanCount(0)
		for rank := 1; rank <= 7; rank++ {
			cur := KoanCount(rank)
			if cur <= prev {
				t.Errorf("KoanCount(%d) = %d, not greater than KoanCount(%d) = %d",
					rank, cur, rank-1, prev)
			}
			prev = cur
		}
	})
}

// ─── Belt Quotes ─────────────────────────────────────────────────────────────

func TestBeltQuote(t *testing.T) {
	for rank := 1; rank <= 7; rank++ {
		t.Run(Belts[rank].Name, func(t *testing.T) {
			q := BeltQuote(rank)
			if q == "" {
				t.Errorf("BeltQuote(%d) returned empty string", rank)
			}
			if q == "The journey continues." {
				t.Errorf("BeltQuote(%d) returned fallback, expected a real quote", rank)
			}
		})
	}

	t.Run("rank 0 returns fallback", func(t *testing.T) {
		q := BeltQuote(0)
		if q != "The journey continues." {
			t.Errorf("BeltQuote(0) = %q, want fallback", q)
		}
	})

	t.Run("invalid rank returns fallback", func(t *testing.T) {
		q := BeltQuote(99)
		if q != "The journey continues." {
			t.Errorf("BeltQuote(99) = %q, want fallback", q)
		}
	})
}
