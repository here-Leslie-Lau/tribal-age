package tui

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"tribal-age/internal/game"
)

func colorModel(t *testing.T, s *game.State, profile colorprofile.Profile) *model {
	t.Helper()
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	m := newModel(s)
	m.Update(tea.ColorProfileMsg{Profile: profile})
	return m
}

func ownershipState() *game.State {
	s := productionState()
	p := s.Tribes[0].Capital
	for i, owner := range []int{1, 2, game.Unowned} {
		q := game.Point{X: p.X + i, Y: p.Y + 1}
		s.Tiles[q.Y][q.X].Owner = owner
		if owner != game.Unowned {
			s.Tribes[owner].Explored[q.Y][q.X] = true
		}
	}
	return s
}

func mapCell(m *model, p game.Point) string {
	l := screenLayout(m.width, m.height, m.cursor)
	row := 4 + p.Y - l.top
	if l.roomy {
		row += 2
	}
	left := l.inset + 4 + (p.X-l.left)*mapCellWidth
	return ansi.Cut(strings.Split(m.View().Content, "\n")[row], left, left+mapCellWidth)
}

func background(cell string) string {
	// Cut preserves later SGR codes too; inspect the style before visible text.
	prefix := regexp.MustCompile(`^(?:\x1b\[[0-9;]*m)*`).FindString(cell)
	matches := regexp.MustCompile(`48;2;[0-9]+;[0-9]+;[0-9]+`).FindAllString(prefix, -1)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1]
}

func TestTerritoryColorConnectsTerrainAndSeparatesOwners(t *testing.T) {
	s := ownershipState()
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	m := colorModel(t, s, colorprofile.TrueColor)
	m.cursor = game.Point{}
	before := snapshot(s)
	p := s.Tribes[0].Capital
	grass := mapCell(m, game.Point{X: p.X + 1, Y: p.Y})
	forest := mapCell(m, game.Point{X: p.X + 2, Y: p.Y})
	if ansi.Strip(grass) != " .  " || ansi.Strip(forest) != " f  " {
		t.Fatal("colored tiles still require owner characters or lack full-cell padding")
	}
	if background(grass) == "" || background(grass) != background(forest) {
		t.Fatal("same territory did not connect across terrain types")
	}
	seen := map[string]bool{background(grass): true}
	for i := 0; i < 3; i++ {
		cell := mapCell(m, game.Point{X: p.X + i, Y: p.Y + 1})
		bg := background(cell)
		if bg == "" || seen[bg] || ansi.Strip(cell) != " f  " {
			t.Fatal("foreign or unclaimed territory lacks distinct background and plain terrain glyph")
		}
		seen[bg] = true
	}
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("color rendering mutated the game")
	}
}

func TestCursorUsesFrameOnCapitalEnemyAndFog(t *testing.T) {
	for _, profile := range []colorprofile.Profile{colorprofile.ASCII, colorprofile.TrueColor} {
		m := colorModel(t, ownershipState(), profile)
		p := m.cursor
		want := []string{"[.@]", "[f2]", "[??]"}
		if profile == colorprofile.TrueColor {
			want = []string{"[@] ", "[f] ", "[?] "}
		}
		if got := ansi.Strip(mapCell(m, p)); got != want[0] {
			t.Fatalf("capital cursor %q, want %q", got, want[0])
		}
		press(m, "down")
		if got := ansi.Strip(mapCell(m, m.cursor)); got != want[1] {
			t.Fatalf("enemy cursor %q, want %q", got, want[1])
		}
		for i := 0; i < 8; i++ {
			press(m, "up", "left")
		}
		if got := ansi.Strip(mapCell(m, m.cursor)); got != want[2] {
			t.Fatalf("fog cursor %q, want %q", got, want[2])
		}
	}
}

func TestColorProfilesAndForcedPlainMap(t *testing.T) {
	for _, profile := range []colorprofile.Profile{colorprofile.Unknown, colorprofile.NoTTY, colorprofile.ASCII, colorprofile.ANSI, colorprofile.ANSI256, colorprofile.TrueColor} {
		m := colorModel(t, game.New(42), profile)
		content := m.View().Content
		colored := profile >= colorprofile.ANSI
		if strings.Contains(content, "\x1b[") != colored {
			t.Fatalf("wrong color support for %v", profile)
		}
		if !colored && (!strings.Contains(content, "[.@]") || !strings.Contains(content, "无色")) {
			t.Fatal("plain map lost ownership or cursor")
		}
		if profile == colorprofile.ANSI && (strings.Contains(content, ";2;") || strings.Contains(content, ";5;")) {
			t.Fatal("16-color terminal received RGB or indexed colors")
		}
		if profile == colorprofile.ANSI256 && strings.Contains(content, ";2;") {
			t.Fatal("256-color terminal received RGB")
		}
	}
	for _, env := range []struct{ key, value string }{{"NO_COLOR", "1"}, {"TERM", "dumb"}} {
		t.Run(env.key, func(t *testing.T) {
			t.Setenv(env.key, env.value)
			m := newModel(game.New(42))
			m.Update(tea.ColorProfileMsg{Profile: colorprofile.TrueColor})
			press(m, "v")
			if strings.Contains(m.View().Content, "\x1b[") || !strings.Contains(m.View().Content, "[.@]") {
				t.Fatal("forced plain mode lost characters or accepted color upgrade")
			}
		})
	}
}

func TestColoredFogNeverRevealsHiddenOwnerOrStocks(t *testing.T) {
	a, b := game.New(42), game.New(42)
	b.Tiles[0][0] = game.Tile{Terrain: game.Grassland, Owner: 1, Farm: game.FarmReady}
	b.Tribes[1].Explored[0][0] = true
	b.Tribes[1].Food, b.Tribes[1].Wood, b.Tribes[1].Population = 999, 888, 55
	first, second := colorModel(t, a, colorprofile.TrueColor), colorModel(t, b, colorprofile.TrueColor)
	first.cursor, second.cursor = game.Point{}, game.Point{}
	for i := 0; i < 2; i++ {
		if first.View().Content != second.View().Content {
			t.Fatal("unknown ownership, terrain, border or private stock leaked")
		}
		press(first, "v")
		press(second, "v")
	}
}
