package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"tribal-age/internal/game"
)

func TestSimulationIsReproducible(t *testing.T) {
	var first, second bytes.Buffer
	args := []string{"simulate", "--seed", "42", "--months", "120"}
	if err := Run(args, strings.NewReader(""), &first); err != nil {
		t.Fatal(err)
	}
	if err := Run(args, strings.NewReader("ignored input"), &second); err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() {
		t.Fatal("same seed produced different output")
	}
	for _, text := range []string{"实际推进 120 个月", "第 11 年 3 月", "部落1", "部落2", "部落3", "结算摘要"} {
		if !strings.Contains(first.String(), text) {
			t.Fatalf("missing %q: %s", text, first.String())
		}
	}
	if strings.Contains(first.String(), "获胜") {
		t.Fatal("simulation invented a winner")
	}
}

func TestInteractiveActions(t *testing.T) {
	s := game.New(42)
	p := s.Tribes[0].Capital
	input := fmt.Sprintf("help\nmap\nstatus\nfarm %d %d\nfarm %d %d\ntrain\ntrain\nend\nstatus\nquit\n", p.X, p.Y, p.X, p.Y)
	var output bytes.Buffer
	if err := play(s, strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	if s.Calendar.Elapsed != 1 || s.Tribes[0].Food != 47 || s.Tribes[0].Wood != 10 || s.Tribes[0].Soldiers != 10 || s.Tribes[0].AP != 2 {
		t.Fatal("CLI applied the wrong rules or repeated an action")
	}
	for _, text := range []string{"已有在建或完工农田", "行动点不足", "第 1 年 4 月", "1/3", "上月净变化", "??", "图例"} {
		if !strings.Contains(output.String(), text) {
			t.Fatalf("missing %q", text)
		}
	}
}

func TestReadOnlyCommandsAndEOFDoNotAdvance(t *testing.T) {
	s := game.New(42)
	var output bytes.Buffer
	if err := play(s, strings.NewReader("help\nmap\nstatus\n\n"), &output); err != nil {
		t.Fatal(err)
	}
	if s.Calendar.Elapsed != 0 || s.Tribes[0].AP != 2 || len(s.Events) != 0 {
		t.Fatal("time advanced without end")
	}
}

func TestCLIRejectsMalformedInput(t *testing.T) {
	for _, args := range [][]string{
		{"unknown"}, {"play", "--bad"}, {"play", "extra"}, {"simulate", "--months", "-1"},
		{"simulate", "--months", "oops"}, {"play", "--seed", "oops"}, {"play", "--seed", "99999999999999999999999999999"},
	} {
		var out bytes.Buffer
		if err := Run(args, strings.NewReader(""), &out); err == nil {
			t.Fatalf("accepted args %v", args)
		}
	}
	for _, input := range []string{"unknown", "farm", "farm 1", "farm x 2", "farm 1 y", "farm 1 2 extra", "train extra", "end extra", "map extra", "explore -1 0", "claim 24 0", "farm 9999999999999999999999999 0"} {
		s := game.New(42)
		var out bytes.Buffer
		if err := play(s, strings.NewReader(input+"\nquit\n"), &out); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "错误") || s.Calendar.Elapsed != 0 || s.Tribes[0].AP != 2 || s.Tribes[0].Wood != 20 {
			t.Fatalf("bad input mutated state: %q", input)
		}
	}
}

func TestHelpAndZeroMonthSimulation(t *testing.T) {
	for _, args := range [][]string{nil, {"help"}, {"--help"}, {"play", "--help"}, {"simulate", "--help"}, {"simulate", "--months", "0"}} {
		var out bytes.Buffer
		if err := Run(args, strings.NewReader(""), &out); err != nil || out.Len() == 0 {
			t.Fatalf("args %v: %v", args, err)
		}
	}
}

func TestPlayHidesUnknownWorldAndEnemyInventory(t *testing.T) {
	a, b := game.New(42), game.New(42)
	b.Tiles[0][0] = game.Tile{Terrain: game.Forest, Owner: 1}
	b.Tribes[1].Explored[0][0] = true
	b.Tribes[1].Food = 999
	b.Tribes[1].Wood = 888
	b.Tribes[1].Population = 55
	b.Tribes[1].HungerMonths = 1
	b.Tribes[1].ForestPriority = 4
	var first, second bytes.Buffer
	if err := play(a, strings.NewReader("map\nstatus\nquit\n"), &first); err != nil {
		t.Fatal(err)
	}
	if err := play(b, strings.NewReader("map\nstatus\nquit\n"), &second); err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() {
		t.Fatal("hidden world or enemy inventory leaked into CLI")
	}
}

func TestExtinctionStopsAtActualMonth(t *testing.T) {
	t.Run("player", func(t *testing.T) {
		s := game.New(42)
		s.Tribes[0].Population = 1
		s.Tribes[0].Soldiers = 1
		s.Tribes[0].Food = 0
		s.Tribes[0].Wood = 0
		var out bytes.Buffer
		if err := play(s, strings.NewReader("end\nend\nend\n"), &out); err != nil {
			t.Fatal(err)
		}
		if s.Calendar.Elapsed != 2 || !strings.Contains(out.String(), "生存结束") {
			t.Fatal("dead player's game continued")
		}
		if err := s.Validate(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("all tribes", func(t *testing.T) {
		s := game.New(42)
		for id := range s.Tribes {
			s.Tribes[id].Population = 1
			s.Tribes[id].Soldiers = 1
			s.Tribes[id].Food = 0
			s.Tribes[id].Wood = 0
		}
		var out bytes.Buffer
		if err := simulate(s, 120, &out); err != nil {
			t.Fatal(err)
		}
		if s.Calendar.Elapsed != 2 || !strings.Contains(out.String(), "实际推进 2 个月") || !strings.Contains(out.String(), "灭亡") {
			t.Fatal("wrong early termination summary")
		}
	})
}

var errIO = errors.New("test I/O failure")

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, errIO }

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errIO }

func TestIOErrorsAreReturned(t *testing.T) {
	if err := Run([]string{"play"}, brokenReader{}, io.Discard); !errors.Is(err, errIO) {
		t.Fatalf("lost read error: %v", err)
	}
	if err := Run([]string{"play"}, strings.NewReader(""), brokenWriter{}); !errors.Is(err, errIO) {
		t.Fatalf("lost write error: %v", err)
	}
	if err := Run([]string{"simulate", "--months", "0"}, strings.NewReader(""), brokenWriter{}); !errors.Is(err, errIO) {
		t.Fatalf("lost write error: %v", err)
	}
	if err := Run([]string{"play"}, strings.NewReader(strings.Repeat("x", 100000)+"\n"), io.Discard); err == nil {
		t.Fatal("oversized input silently ignored")
	}
}
