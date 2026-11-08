package tui

import (
	"strings"
	"testing"
	"tribal-age/internal/game"
)

func battleAndCutoffModel() *model {
	m := cutoffModel()
	s := m.state
	p := s.Tribes[0].Capital
	q := game.Point{X: p.X + 1, Y: p.Y}
	s.Tiles[p.Y][p.X].Terrain = game.Grassland
	s.Tiles[q.Y][q.X] = game.Tile{Terrain: game.Grassland, Owner: 1}
	s.Tribes[1].Capital = q
	s.Tribes[1].Explored[q.Y][q.X] = true
	s.Tribes[1].Explored[p.Y][p.X] = true
	s.Tribes[0].Explored[q.Y][q.X] = true
	s.Tribes[0].Soldiers = 21
	s.Tribes[1].Soldiers = 15
	s.Tribes[0].Armies[2] = s.Tribes[0].Armies[0]
	s.Tribes[0].Armies[0] = game.Army{People: 20, Position: p, Order: game.ArmyOrder{Kind: game.Attack, Target: q}}
	s.Tribes[1].Armies[0] = game.Army{People: 15, Position: q, Order: game.ArmyOrder{Kind: game.Attack, Target: p}}
	return m
}
func TestFatalArmyCauseIsVisibleBeforeLongBattleDetails(t *testing.T) {
	m := battleAndCutoffModel()
	press(m, "enter", "y")
	if m.dialog != reportDialog {
		t.Fatal("fatal monthly report did not open")
	}
	text := m.View().Content
	if !strings.Contains(text, "军队C") || !strings.Contains(text, "持续断供") {
		t.Fatal("long battle details hid the reason an army disappeared on 80x24")
	}
	if strings.Index(text, "持续断供") > strings.Index(text, "交锋1：") {
		t.Fatal("critical disappearance reason must precede battle details")
	}
}
