package tui

import (
	"reflect"
	"strings"
	"testing"

	"tribal-age/internal/game"
)

func TestAttackPreviewIsConditionalAndReadOnly(t *testing.T) {
	s := game.New(42)
	p := s.Tribes[0].Capital
	target := game.Point{X: p.X, Y: p.Y - 1}
	s.Tribes[0].Armies[0] = game.Army{People: 5, Position: p}
	s.Tribes[1].Armies[0] = game.Army{People: 5, Position: target}
	s.Tribes[0].Explored[target.Y][target.X], s.Tribes[1].Explored[target.Y][target.X] = true, true
	m := newModel(s)
	press(m, "1", "up", "f")
	before := snapshot(s)
	text := m.View().Content
	if !strings.Contains(text, "估损我1敌1") || !strings.Contains(text, "未计其他军令") || !reflect.DeepEqual(before, *s) {
		t.Fatal("attack estimate was missing, unconditional or mutated the game")
	}
}
