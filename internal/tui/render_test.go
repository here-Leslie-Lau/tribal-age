package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"tribal-age/internal/game"
)

func TestScreenKeepsMapStateActionsAndLegendTogether(t *testing.T) {
	s := game.New(42)
	m := newModel(s)
	before := snapshot(s)
	v := m.View()
	for _, text := range []string{"部落纪元", "第 1 年 3 月", "行动点：2/2", "人口：30/30", "粮食：60", "木材：20", ">.@", "??", "选中", "农田", "10木", "WASD", "P生产", "Enter", "Q退出", "@首都", "f森林"} {
		if !strings.Contains(v.Content, text) {
			t.Fatalf("missing %q:\n%s", text, v.Content)
		}
	}
	if !v.AltScreen || v.MouseMode != tea.MouseModeNone {
		t.Fatal("screen requires mouse or is not full-screen")
	}
	if !reflect.DeepEqual(before, *s) || m.View().Content != v.Content {
		t.Fatal("rendering mutated state or was nondeterministic")
	}
}

func TestScreenFitsTerminalAndTracksCursorViewport(t *testing.T) {
	m := newModel(game.New(42))
	for _, size := range [][2]int{{80, 24}, {80, 32}, {100, 24}, {100, 30}, {99, 32}, {100, 31}, {100, 32}, {100, 33}, {160, 48}, {100000, 100000}, {79, 24}, {80, 23}, {40, 12}, {0, 0}, {-1, -1}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		content := m.View().Content
		if content == "" {
			continue
		}
		lines := strings.Split(content, "\n")
		if len(lines) > max(0, size[1]) {
			t.Fatalf("screen height overflow for %v: %d", size, len(lines))
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > max(0, size[0]) {
				t.Fatalf("screen width overflow for %v: %q", size, line)
			}
		}
		if size[0] < 80 && size[0] > 20 && !strings.Contains(content, "终端") {
			t.Fatal("small terminal lacks an explanation")
		}
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	for i := 0; i < 30; i++ {
		press(m, "right", "down")
	}
	content := m.View().Content
	if !strings.Contains(content, "选中 (23,15)") || !strings.Contains(content, ">??") || !strings.Contains(content, "未探索") {
		t.Fatalf("viewport lost cursor at map edge:\n%s", content)
	}
}

func TestRenderDoesNotRevealUnknownTilesOrEnemyStocks(t *testing.T) {
	a, b := game.New(42), game.New(42)
	b.Tiles[0][0] = game.Tile{Terrain: game.Grassland, Owner: 1, Farm: game.FarmReady}
	b.Tribes[1].Explored[0][0] = true
	b.Tribes[1].Food, b.Tribes[1].Wood, b.Tribes[1].Population, b.Tribes[1].ForestPriority = 999, 888, 55, 4
	first, second := newModel(a), newModel(b)
	first.cursor, second.cursor = game.Point{}, game.Point{}
	if first.View().Content != second.View().Content || !strings.Contains(first.View().Content, "未探索") {
		t.Fatal("unknown tile or enemy inventory leaked")
	}
	p := a.Tribes[1].Capital
	a.Tribes[0].Explored[p.Y][p.X], b.Tribes[0].Explored[p.Y][p.X] = true, true
	first.cursor, second.cursor = p, p
	if first.View().Content != second.View().Content || !strings.Contains(first.View().Content, "部落2") {
		t.Fatal("known enemy tile exposed private data")
	}
}

func productionState() *game.State {
	s := game.New(42)
	s.Calendar.Elapsed = 3
	s.Tribes[0].Population, s.Tribes[0].Food = 31, 30
	p := s.Tribes[0].Capital
	for i := 0; i < 6; i++ {
		q := game.Point{X: p.X + i%3, Y: p.Y + i/3}
		tile := game.Tile{Terrain: game.Forest, Owner: 0}
		if i < 2 {
			tile.Terrain, tile.Farm = game.Grassland, game.FarmReady
		}
		s.Tiles[q.Y][q.X] = tile
		s.Tribes[0].Explored[q.Y][q.X] = true
	}
	return s
}

func TestProductionDraftPreviewsTradeoffWithoutCommitting(t *testing.T) {
	s := productionState()
	m := newModel(s)
	if !strings.Contains(m.View().Content, "收支 +4") {
		t.Fatal("automatic production forecast is missing")
	}
	before := snapshot(s)
	press(m, "p", "right", "right", "right", "right")
	content := m.View().Content
	for _, text := range []string{"草稿", "收支 -2", "预计 +8", "余粮 28"} {
		if !strings.Contains(content, text) {
			t.Fatalf("missing draft forecast %q:\n%s", text, content)
		}
	}
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("draft preview changed stocks or staffing")
	}
	press(m, "enter", "enter", "y")
	if s.Tribes[0].Food != 28 || s.Tribes[0].Wood != 28 {
		t.Fatal("draft forecast differed from actual settlement")
	}
}

func TestMenusExplainCostsRiskAndCancellation(t *testing.T) {
	m := newModel(productionState())
	for _, tt := range []struct{ key, text string }{{"t", "5粮"}, {"p", "草稿"}, {"enter", "结束本月"}, {"?", "帮助"}, {"q", "存档"}} {
		press(m, tt.key)
		if !strings.Contains(m.View().Content, tt.text) {
			t.Fatalf("menu %q hid %q", tt.key, tt.text)
		}
		press(m, "esc")
	}
	m.state.Tribes[0].Food = 1
	press(m, "p", "right", "right", "right", "right")
	if !strings.Contains(m.View().Content, "缺粮预警") {
		t.Fatal("manual staffing did not warn about hunger")
	}
}

func TestScreenShowsNewlyCompletingFarmAndNoAvailableActions(t *testing.T) {
	s := game.New(42)
	p := s.Tribes[0].Capital
	s.Tiles[p.Y][p.X].Farm = game.FarmReady - 1
	s.Tribes[0].AP = 0
	m := newModel(s)
	content := m.View().Content
	if !strings.Contains(content, "收支 -3") || !strings.Contains(content, "暂无可用操作") || !strings.Contains(content, "2/3") {
		t.Fatalf("completion or unavailable action feedback is wrong:\n%s", content)
	}
}
