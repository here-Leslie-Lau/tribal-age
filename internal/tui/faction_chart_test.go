package tui

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"tribal-age/internal/game"
)

func chartState(t *testing.T, counts [game.TribeCount]int) *game.State {
	t.Helper()
	s := game.New(42)
	for _, tribe := range s.Tribes {
		s.Tribes[0].Explored[tribe.Capital.Y][tribe.Capital.X] = true
	}
	for id, target := range counts {
		remaining := target - 1
		for y := 1; y < game.Height-1; y++ {
			for x := 1; x < game.Width-1; x++ {
				if remaining == 0 || s.Tiles[y][x].Owner != game.Unowned {
					continue
				}
				s.Tiles[y][x] = game.Tile{Terrain: game.Grassland, Owner: id}
				s.Tribes[id].Explored[y][x], s.Tribes[0].Explored[y][x] = true, true
				remaining--
			}
		}
		if remaining != 0 {
			t.Fatal("chart fixture exhausted map")
		}
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	return s
}

func chartRow(m *model, label string) string {
	l := screenLayout(m.width, m.height, m.cursor)
	left := l.inset + l.leftWidth + len(l.gap)
	for _, line := range strings.Split(m.View().Content, "\n") {
		text := ansi.Strip(ansi.Cut(line, left, left+l.rightWidth))
		if strings.Contains(text, label) {
			return text
		}
	}
	return ""
}

func TestFactionChartShowsProportionalASCIIAndExactKnownCounts(t *testing.T) {
	for _, counts := range [][game.TribeCount]int{{12, 8, 5}, {120, 80, 50}, {280, 1, 1}} {
		s := chartState(t, counts)
		m := colorModel(t, s, colorprofile.TrueColor)
		m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
		before := snapshot(s)
		if !strings.Contains(ansi.Strip(m.View().Content), "大陆局势（已知）") {
			t.Fatal("expanded sidebar lacks known-territory chart")
		}
		bars := []int{strings.Count(chartRow(m, "我方"), "#"), strings.Count(chartRow(m, "部落2"), "#"), strings.Count(chartRow(m, "部落3"), "#")}
		if bars[0] != 12 {
			t.Fatalf("largest known faction should fill the bounded bar: %v", bars)
		}
		if counts[0] < 280 && (bars[1] != 8 || bars[2] != 5) {
			t.Fatalf("bar proportions are wrong for %v: %v", counts, bars)
		}
		if counts[0] == 280 && (bars[1] != 1 || bars[2] != 1) {
			t.Fatal("small known factions vanished from chart")
		}
		for i, label := range []string{"我方", "部落2", "部落3"} {
			if !strings.Contains(chartRow(m, label), strconv.Itoa(counts[i])+"格") {
				t.Fatal("scaled chart lost exact count")
			}
		}
		if !reflect.DeepEqual(before, *s) {
			t.Fatal("chart changed game state")
		}
	}
}

func TestFactionChartUnknownAndEmptyLandAreSafe(t *testing.T) {
	s := game.New(42)
	s.Tribes[0].Population, s.Tribes[0].Soldiers, s.Tribes[0].Food = 1, 1, 0
	for i := 0; i < 2; i++ {
		if err := s.EndMonth(game.Interactive); err != nil {
			t.Fatal(err)
		}
	}
	m := colorModel(t, s, colorprofile.ASCII)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 32})
	if !strings.Contains(chartRow(m, "我方"), "0格") || strings.Count(chartRow(m, "我方"), "#") != 0 {
		t.Fatal("empty known territory chart is misleading")
	}
	for _, label := range []string{"部落2", "部落3"} {
		row := chartRow(m, label)
		if !strings.Contains(row, "未知") || !strings.Contains(row, "?") || strings.Contains(row, "#") || strings.Contains(row, "0格") {
			t.Fatal("unknown faction was drawn as a known total")
		}
	}
}

func TestFactionChartFitsNarrowRoomyAndPlainScreens(t *testing.T) {
	for _, profile := range []colorprofile.Profile{colorprofile.ASCII, colorprofile.TrueColor} {
		m := colorModel(t, chartState(t, [game.TribeCount]int{120, 80, 50}), profile)
		for _, size := range [][2]int{{80, 24}, {80, 32}, {100, 32}, {120, 36}, {160, 48}} {
			m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			content := m.View().Content
			if len(strings.Split(content, "\n")) > size[1] {
				t.Fatal("chart exceeded height")
			}
			for _, line := range strings.Split(content, "\n") {
				if ansi.StringWidth(line) > size[0] {
					t.Fatal("chart exceeded width")
				}
			}
			if size[1] < 32 {
				if strings.Contains(content, "大陆局势") || !strings.Contains(content, "已知领土 我120 2:80 3:50") {
					t.Fatal("compact screen lost readable totals")
				}
			} else if strings.Count(chartRow(m, "我方"), "#") == 0 {
				t.Fatal("roomy chart was lost")
			}
		}
	}
}

func TestFactionChartDoesNotUseHiddenLandForCountsOrScale(t *testing.T) {
	a, b := chartState(t, [game.TribeCount]int{12, 8, 5}), chartState(t, [game.TribeCount]int{12, 8, 5})
	b.Tiles[0][0] = game.Tile{Terrain: game.Forest, Owner: 1}
	b.Tribes[1].Explored[0][0] = true
	b.Tribes[1].Food, b.Tribes[1].Population = 999, 55
	first, second := colorModel(t, a, colorprofile.TrueColor), colorModel(t, b, colorprofile.TrueColor)
	for _, m := range []*model{first, second} {
		m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	}
	if first.View().Content != second.View().Content {
		t.Fatal("hidden expansion or enemy private state changed chart")
	}
}
