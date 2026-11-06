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

func TestArmyManagementAndOrdersAreSafeAtAllSizes(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {80, 32}, {120, 36}, {160, 48}} {
		s := game.New(42)
		m := newModel(s)
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		before := snapshot(s)
		press(m, "m", "2", "right", "left", "home", "3", "+", "-", "0", "esc")
		if !reflect.DeepEqual(before, *s) {
			t.Fatal("army menu browsing changed resources or time")
		}
		press(m, "m", "1", "enter", "h", "o")
		if !strings.Contains(strings.Join(m.messages, "\n"), "只能接收") {
			t.Fatal("invalid occupation lacked an engine reason")
		}
		press(m, "m", "1")
		text := m.View().Content
		for _, want := range []string{"军队A：5人", "军队B：未编组", "军队C：未编组", "首都", "军令月底执行"} {
			if !strings.Contains(text, want) {
				t.Fatalf("size %v lost %s", size, want)
			}
		}
		press(m, "0", "enter")
		if s.Tribes[0].ReserveSoldiers() != 5 || s.Tribes[0].Armies[0].People != 0 {
			t.Fatal("disband was not applied")
		}
	}
}
func TestArmyOccupyControlAndOwnerBackground(t *testing.T) {
	s := game.New(42)
	p := s.Tribes[1].Capital
	s.Tribes[0].Armies[0] = game.Army{People: 5, Position: p}
	s.Tribes[0].Explored[p.Y][p.X] = true
	m := colorModel(t, s, colorprofile.TrueColor)
	press(m, "1")
	cell := mapCell(m, p)
	base := newModel(game.New(42))
	base.mapView = m.mapView
	base.state.Tribes[0].Explored[p.Y][p.X] = true
	base.cursor = p
	if !strings.Contains(ansi.Strip(cell), "A1") || background(cell) != background(mapCell(base, p)) {
		t.Fatal("army marker falsified the underlying territory owner")
	}
	press(m, "o")
	if s.Tiles[p.Y][p.X].Owner != 1 || s.Tribes[0].Armies[0].Order.Kind != game.Occupy {
		t.Fatal("occupation executed before the month ended")
	}
	press(m, "enter", "y")
	if s.Tiles[p.Y][p.X].Owner != 0 || (s.Tribes[1].Alive() && s.Tribes[1].Capital == p) {
		t.Fatal("occupation command failed at month end")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}
