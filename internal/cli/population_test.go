package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"tribal-age/internal/game"
)

func TestStatusExplainsPopulationAndLabor(t *testing.T) {
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
	var out bytes.Buffer
	if err := play(s, strings.NewReader("status\nquit\n"), &out); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"人口：30/55", "增长进度：2/3", "安全存粮：24", "可经营 5 格", "农田 2 格", "森林 3 格", "人手不足"} {
		if !strings.Contains(out.String(), text) {
			t.Fatalf("missing population feedback %q: %s", text, out.String())
		}
	}
}

func TestGrowthAndGraceFeedbackThroughCommands(t *testing.T) {
	t.Run("opening expansion", func(t *testing.T) {
		s := game.New(42)
		p := s.Tribes[0].Capital
		input := fmt.Sprintf("farm %d %d\nexplore %d %d\nend\nclaim %d %d\nend\nend\nend\nquit\n", p.X, p.Y, p.X, p.Y-1, p.X, p.Y-1)
		var out bytes.Buffer
		if err := play(s, strings.NewReader(input), &out); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].Population != 31 || s.Calendar.Elapsed != 4 {
			t.Fatal("expansion did not enable three qualifying months of growth")
		}
		for _, text := range []string{"承载力已满", "增长进度：1/3", "增长进度：2/3", "人口自然增长"} {
			if !strings.Contains(out.String(), text) {
				t.Fatalf("missing feedback %q", text)
			}
		}
	})
	t.Run("grace then starvation", func(t *testing.T) {
		s := game.New(42)
		s.Tribes[0].Food = 0
		s.Tribes[0].Wood = 0
		var out bytes.Buffer
		if err := play(s, strings.NewReader("end\nstatus\nend\nquit\n"), &out); err != nil {
			t.Fatal(err)
		}
		for _, text := range []string{"饥荒宽限", "连续缺粮：1 月", "连续缺粮：2 月", "人口减少 7"} {
			if !strings.Contains(out.String(), text) {
				t.Fatalf("missing hunger feedback %q", text)
			}
		}
	})
}
