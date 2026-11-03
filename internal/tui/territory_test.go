package tui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"tribal-age/internal/game"
)

func TestTerritoryListShowsOwnedFarmsAndCurrentProgress(t *testing.T) {
	s := productionState()
	p := s.Tribes[0].Capital
	s.Tiles[p.Y][p.X].Farm = game.FarmPlanned
	s.Tiles[p.Y][p.X+1].Farm = game.FarmReady - 1
	q := game.Point{X: p.X + 2, Y: p.Y}
	s.Tiles[q.Y][q.X] = game.Tile{Terrain: game.Grassland, Owner: 0, Farm: game.FarmReady}
	m := newModel(s)
	before := snapshot(s)
	press(m, "l")
	content := m.View().Content
	for _, text := range []string{"领地一览", "领地6格", "完工农田1", "在建农田2", "农田建造中 0/3", "农田建造中 2/3", "农田已完工", "森林", "Enter定位", "Esc/L返回"} {
		if !strings.Contains(content, text) {
			t.Fatalf("missing territory feedback %q:\n%s", text, content)
		}
	}
	if !strings.Contains(content, "预计产粮6") {
		t.Fatal("territory list missed the farm completing this month")
	}
	if !reflect.DeepEqual(before, *s) || content != m.View().Content {
		t.Fatal("territory list mutated game state or construction")
	}
	press(m, "esc")
	if strings.Contains(m.View().Content, "领地一览") {
		t.Fatal("territory list did not close")
	}
}

func TestTerritoryNavigationAndLocateDoNotSpendTimeOrAP(t *testing.T) {
	s := productionState()
	m := newModel(s)
	p := s.Tribes[0].Capital
	before := snapshot(s)
	press(m, "l", "down", "enter", "y")
	if m.cursor != (game.Point{X: p.X + 1, Y: p.Y}) || !reflect.DeepEqual(before, *s) {
		t.Fatal("locate did not focus land for free, or Enter ended the month")
	}
	press(m, "l", "b", "c", "e", "t", "p", "y", "l")
	if !reflect.DeepEqual(before, *s) || strings.Contains(m.View().Content, "领地一览") {
		t.Fatal("land viewer allowed game actions or failed to toggle")
	}
}

func TestTerritoryListScrollsAllOwnedLandWithoutOverflow(t *testing.T) {
	s := game.New(42)
	var points []game.Point
	for y := 6; y < 8; y++ {
		for x := 1; x <= 18; x++ {
			p := game.Point{X: x, Y: y}
			if s.Tiles[y][x].Owner != game.Unowned {
				continue
			}
			s.Tiles[y][x] = game.Tile{Terrain: game.Grassland, Owner: 0, Farm: game.FarmReady}
			s.Tribes[0].Explored[y][x] = true
			points = append(points, p)
		}
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	m := newModel(s)
	m.cursor = points[0]
	before := snapshot(s)
	press(m, "l")
	for i := 0; i < 100; i++ {
		press(m, "down")
	}
	content := m.View().Content
	last := points[len(points)-1]
	if !strings.Contains(content, fmt.Sprintf("(%d,%d)", last.X, last.Y)) {
		t.Fatal("last owned tile could not be reached")
	}
	for _, line := range strings.Split(content, "\n") {
		if ansi.StringWidth(line) > 80 {
			t.Fatal("land table overflowed terminal width")
		}
	}
	if len(strings.Split(content, "\n")) > 24 {
		t.Fatal("land table overflowed terminal height")
	}
	press(m, "enter")
	if m.cursor != last || !reflect.DeepEqual(before, *s) {
		t.Fatal("scrolling or locating changed the game")
	}
}

func TestTerritoryListContainsNoForeignOrUnclaimedTiles(t *testing.T) {
	s := productionState()
	foreign := s.Tribes[1].Capital
	s.Tribes[0].Explored[foreign.Y][foreign.X] = true
	s.Tribes[0].Explored[0][0] = true
	s.Tiles[0][0] = game.Tile{Terrain: game.Forest, Owner: game.Unowned}
	m := newModel(s)
	press(m, "l")
	content := m.View().Content
	if strings.Contains(content, fmt.Sprintf("(%d,%d)", foreign.X, foreign.Y)) || strings.Contains(content, "(0,0)") {
		t.Fatal("land viewer included foreign or unclaimed plots")
	}
	s.Tribes[1].Food, s.Tribes[1].Wood, s.Tribes[1].ForestPriority = 999, 888, 4
	if m.View().Content != content {
		t.Fatal("enemy private state leaked into land viewer")
	}
}

func TestEmptyTerritoryListAndResizeAreSafe(t *testing.T) {
	s := game.New(42)
	s.Tribes[0].Population, s.Tribes[0].Soldiers, s.Tribes[0].Food = 1, 1, 0
	if err := s.EndMonth(game.Interactive); err != nil {
		t.Fatal(err)
	}
	if err := s.EndMonth(game.Interactive); err != nil {
		t.Fatal(err)
	}
	m := newModel(s)
	before := snapshot(s)
	press(m, "l", "up", "down")
	if !strings.Contains(m.View().Content, "暂无领地") {
		t.Fatal("empty land list lacks explanation")
	}
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	press(m, "enter")
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	if !strings.Contains(m.View().Content, "暂无领地") {
		t.Fatal("resizing lost safe empty-list state")
	}
	press(m, "enter")
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("empty list navigation revived or advanced the tribe")
	}
}

func TestMapHasBreathingRoomWithoutBreakingCompactScreen(t *testing.T) {
	m := newModel(game.New(42))
	compact := m.View().Content
	if !strings.Contains(compact, " ??  ?? ") {
		t.Fatal("map cells remain packed together")
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	content := m.View().Content
	lines := strings.Split(content, "\n")
	if strings.TrimSpace(lines[1]) != "" || !strings.Contains(content, "   |   ") {
		t.Fatal("large screen lacks header and column separation")
	}
	for _, text := range []string{"人口：30/30", "粮食：60", "木材：20", "行动点：2/2", "L领地", "增长", "Q退出"} {
		if !strings.Contains(content, text) {
			t.Fatalf("spacing hid %q", text)
		}
	}
	for _, line := range lines {
		if ansi.StringWidth(line) > 120 {
			t.Fatal("roomy screen overflowed")
		}
	}
}

func TestTerritoryFarmProgressUpdatesOnlyAfterMonthEnd(t *testing.T) {
	s := game.New(42)
	m := newModel(s)
	press(m, "b")
	for _, status := range []string{"农田建造中 0/3", "农田建造中 1/3", "农田建造中 2/3", "农田已完工"} {
		before := snapshot(s)
		press(m, "l")
		if !strings.Contains(m.View().Content, status) {
			t.Fatalf("land list missed current status %q", status)
		}
		press(m, "enter")
		if !reflect.DeepEqual(before, *s) {
			t.Fatal("viewing progress advanced construction")
		}
		press(m, "enter", "y")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestTerritoryListShowsUnstaffedFarmAndWinterRest(t *testing.T) {
	s := productionState()
	if err := s.SetForestPriority(0, 4); err != nil {
		t.Fatal(err)
	}
	m := newModel(s)
	press(m, "l")
	p := s.Tribes[0].Capital
	position := fmt.Sprintf("(%d,%d)", p.X+1, p.Y)
	found := false
	for _, line := range strings.Split(m.View().Content, "\n") {
		if strings.Contains(line, position) && strings.Contains(line, "人手不足") {
			found = true
		}
	}
	if !found {
		t.Fatal("the specific unstaffed farm was not explained")
	}
	press(m, "down", "down", "home", "enter")
	if m.cursor != p {
		t.Fatal("Home did not return to the first owned plot")
	}
	s.Calendar.Elapsed = 9
	press(m, "l")
	if !strings.Contains(m.View().Content, "冬季休耕") || !strings.Contains(m.View().Content, "预计采木2") {
		t.Fatal("land list did not show winter forestry and resting fields")
	}
}
