package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"tribal-age/internal/game"
)

func TestFactionSummaryCountsKnownLandBeyondViewport(t *testing.T) {
	s := ownershipState()
	p := game.Point{X: 20, Y: 14}
	s.Tiles[p.Y][p.X] = game.Tile{Terrain: game.Forest, Owner: 1}
	s.Tribes[0].Explored[p.Y][p.X], s.Tribes[1].Explored[p.Y][p.X] = true, true
	m := colorModel(t, s, colorprofile.TrueColor)
	before := snapshot(s)
	if !strings.Contains(ansi.Strip(m.View().Content), "已知领土 我3 2:2 3:1") {
		t.Fatal("known land outside viewport was missing or hidden enemy land was counted")
	}
	for i := 0; i < 24; i++ {
		press(m, "right", "down")
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "已知领土 我3 2:2 3:1") || !reflect.DeepEqual(before, *s) {
		t.Fatal("scrolling changed known totals or the game")
	}
}

func TestUnseenFactionsAreNotReportedAsExtinct(t *testing.T) {
	m := colorModel(t, game.New(42), colorprofile.TrueColor)
	if !strings.Contains(ansi.Strip(m.View().Content), "已知领土 我1 2:? 3:?") {
		t.Fatal("unseen factions need unknown, not zero-total labels")
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	content := ansi.Strip(m.View().Content)
	for _, text := range []string{"我方1格", "部落2 暂无已知领地", "部落3 暂无已知领地"} {
		if !strings.Contains(content, text) {
			t.Fatalf("missing unknown-faction explanation %q", text)
		}
	}
}

func sideText(m *model) string {
	l := screenLayout(m.width, m.height, m.cursor)
	start := 2
	if l.roomy {
		start += 2
	}
	left := l.inset + l.leftWidth + len(l.gap)
	lines := strings.Split(m.View().Content, "\n")
	var out strings.Builder
	for _, line := range lines[start : start+l.rows+2] {
		out.WriteString(strings.TrimSpace(ansi.Strip(ansi.Cut(line, left, left+l.rightWidth))))
	}
	return out.String()
}

func TestFactionStatsKeepEssentialSidebarFeedbackAtEverySize(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {80, 32}, {100, 24}, {100, 32}, {120, 36}, {160, 48}} {
		s := ownershipState()
		s.Tribes[0].Food = 0
		m := colorModel(t, s, colorprofile.TrueColor)
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		v, _ := s.View(0)
		f := game.ForecastFor(v)
		text := sideText(m)
		for _, want := range []string{"人口：31/40", "粮食：0", "木材：20", "人手5格", "安全粮24", "连续缺粮：0月", "增长：" + f.Growth.Reason} {
			if !strings.Contains(text, want) {
				t.Fatalf("size %v clipped %q:\n%s", size, want, text)
			}
		}
		if size[1] >= 32 {
			for _, want := range []string{"我方3格", "部落2 已发现1格", "部落3 已发现1格"} {
				if !strings.Contains(text, want) {
					t.Fatalf("size %v clipped expanded stats %q", size, want)
				}
			}
		}
	}
}
