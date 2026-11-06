package tui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"tribal-age/internal/game"
)

func TestArmyPanelDraftAndOrdersUseEngine(t *testing.T) {
	s := game.New(42)
	m := newModel(s)
	before := snapshot(s)
	press(m, "m", "+", "esc")
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("army draft changed the game")
	}
	press(m, "m")
	if !strings.Contains(m.View().Content, "军队管理") {
		t.Fatal("missing army manager")
	}
	press(m, "enter")
	if s.Tribes[0].Armies[0].People != 5 || s.Tribes[0].AP != 1 {
		t.Fatal("army was not assembled through the shared rule")
	}
	p := s.Tribes[0].Capital
	press(m, "1", "up", "f")
	if s.Tribes[0].Armies[0].Position != p || s.Tribes[0].Armies[0].Order.Kind != game.March || s.Tribes[0].AP != 1 {
		t.Fatal("an order executed immediately or spent economic AP")
	}
	if !strings.Contains(m.View().Content, "月底") {
		t.Fatal("queued command lacked feedback")
	}
	press(m, "enter", "y")
	if s.Tribes[0].Armies[0].Position != (game.Point{X: p.X, Y: p.Y - 1}) {
		t.Fatal("end month did not execute the military order")
	}
	press(m, "1")
	if !strings.Contains(ansi.Strip(mapCell(m, s.Tribes[0].Armies[0].Position)), "A1") {
		t.Fatal("map did not show army identity independently of land ownership")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestRememberedTerrainDoesNotRevealArmyFlag(t *testing.T) {
	s := game.New(42)
	p := s.Tribes[1].Capital
	s.Tribes[1].Armies[0] = game.Army{People: 5, Position: p}
	s.Tribes[0].Explored[p.Y][p.X] = true
	m := newModel(s)
	m.cursor = p
	if strings.Contains(ansi.Strip(mapCell(m, p)), "A2") {
		t.Fatal("old exploration exposed a distant enemy flag")
	}
}
