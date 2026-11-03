package cli

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"tribal-age/internal/game"
)

func cliProductionScenario() *game.State {
	s := game.New(42)
	p := s.Tribes[0].Capital
	s.Tribes[0].Food = 100
	s.Tribes[0].GrowthProgress = 2
	for _, q := range []game.Point{p, {X: p.X + 1, Y: p.Y}} {
		s.Tiles[q.Y][q.X] = game.Tile{Terrain: game.Grassland, Owner: 0, Farm: game.FarmReady}
		s.Tribes[0].Explored[q.Y][q.X] = true
	}
	for _, q := range []game.Point{{X: p.X, Y: p.Y - 1}, {X: p.X + 1, Y: p.Y - 1}, {X: p.X - 1, Y: p.Y}, {X: p.X - 1, Y: p.Y - 1}} {
		s.Tiles[q.Y][q.X] = game.Tile{Terrain: game.Forest, Owner: 0}
		s.Tribes[0].Explored[q.Y][q.X] = true
	}
	return s
}

func TestWorkCommandsPersistAndRestoreAutomaticProduction(t *testing.T) {
	s := cliProductionScenario()
	s.Calendar.Elapsed = 3
	s.Tribes[0].Population, s.Tribes[0].Food, s.Tribes[0].GrowthProgress = 31, 30, 0
	var out bytes.Buffer
	if err := play(s, strings.NewReader("work wood 4\nstatus\nend\nend\nwork auto\nend\nquit\n"), &out); err != nil {
		t.Fatal(err)
	}
	if s.Calendar.Elapsed != 6 || s.Tribes[0].ForestPriority != 0 || s.Tribes[0].Food != 30 || s.Tribes[0].Wood != 42 || s.Tribes[0].Population != 32 {
		t.Fatalf("wrong persistent production through CLI: %+v", s.Tribes[0])
	}
	for _, text := range []string{"work wood", "work auto", "优先采木：4 格森林", "预计产粮 6，产木 8", "粮食收支 -2", "预计结余 28", "生产安排：自动"} {
		if !strings.Contains(out.String(), text) {
			t.Fatalf("missing work feedback %q: %s", text, out.String())
		}
	}
}

func TestWorkCommandsAreFreeAndDoNotProduceImmediately(t *testing.T) {
	s := cliProductionScenario()
	s.Tribes[0].AP = 0
	var out bytes.Buffer
	if err := play(s, strings.NewReader("WORK WOOD 4\nstatus\nquit\n"), &out); err != nil {
		t.Fatal(err)
	}
	if s.Calendar.Elapsed != 0 || s.Tribes[0].AP != 0 || s.Tribes[0].Food != 100 || s.Tribes[0].Wood != 20 || s.Tribes[0].GrowthProgress != 2 || s.Tribes[0].ForestPriority != 4 {
		t.Fatalf("work setting advanced or produced resources: %+v", s.Tribes[0])
	}
	if strings.Contains(out.String(), "错误") {
		t.Fatal("valid work command rejected")
	}
}

func TestMalformedWorkCommandsAreAtomic(t *testing.T) {
	for _, input := range []string{"work", "work wood", "work wood x", "work wood 2 extra", "work auto extra", "work food 2", "work wood -1", "work wood 385", "work wood 9999999999999999999999999"} {
		t.Run(input, func(t *testing.T) {
			s := cliProductionScenario()
			if err := s.SetForestPriority(0, 2); err != nil {
				t.Fatal(err)
			}
			before := *s
			before.Events = append([]game.Event(nil), s.Events...)
			var out bytes.Buffer
			if err := play(s, strings.NewReader(input+"\nquit\n"), &out); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "错误") || !reflect.DeepEqual(before, *s) {
				t.Fatalf("invalid work command changed state: %q", input)
			}
		})
	}
}

func TestWorkPreviewExplainsShortageAndUnavailableForests(t *testing.T) {
	s := game.New(42)
	var out bytes.Buffer
	if err := play(s, strings.NewReader("work wood 4\nquit\n"), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "采木目标暂未满足") {
		t.Fatal("unavailable forests were not explained")
	}
	for _, tt := range []struct {
		stock int
		text  string
	}{{25, "预计增长暂停：存粮不足"}, {1, "缺粮预警：预计缺 1 粮"}} {
		s := cliProductionScenario()
		s.Calendar.Elapsed = 3
		s.Tribes[0].Population, s.Tribes[0].Food, s.Tribes[0].GrowthProgress = 31, tt.stock, 0
		out.Reset()
		if err := play(s, strings.NewReader("work wood 4\nquit\n"), &out); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), tt.text) {
			t.Fatalf("missing %q: %s", tt.text, out.String())
		}
		if s.Tribes[0].HungerMonths != 0 || s.Tribes[0].Food != tt.stock {
			t.Fatal("warning settled hunger before end")
		}
	}
}

func TestWorkPreviewIncludesFarmCompletingThisMonth(t *testing.T) {
	s := game.New(42)
	p := s.Tribes[0].Capital
	s.Tiles[p.Y][p.X].Farm = game.FarmReady - 1
	var out bytes.Buffer
	if err := play(s, strings.NewReader("status\nquit\n"), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "预计产粮 4，产木 0") || s.Tiles[p.Y][p.X].Farm != game.FarmReady-1 {
		t.Fatal("preview missed completion or advanced construction")
	}
}
