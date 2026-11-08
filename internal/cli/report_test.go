package cli

import (
	"bytes"
	"strings"
	"testing"
	"tribal-age/internal/game"
)

func TestLineReportsArePrivateAndReadOnly(t *testing.T) {
	s := game.New(42)
	if err := s.SetArmySize(0, 0, 5); err != nil {
		t.Fatal(err)
	}
	if err := s.SetArmySize(0, 0, 0); err != nil {
		t.Fatal(err)
	}
	before, err := s.MarshalSave()
	if err != nil {
		t.Fatal(err)
	}
	c, err := parseCommand("reports")
	if err != nil {
		t.Fatal(err)
	}
	text, err := execute(s, c)
	if err != nil || !strings.Contains(text, "编组调整（人员未死亡）") || !strings.Contains(text, "5 → 0") {
		t.Fatal("line interface cannot explain army removal")
	}
	if _, err := parseCommand("reports 1"); err == nil {
		t.Fatal("report accepted another tribe selector")
	}
	after, err := s.MarshalSave()
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("history command changed game")
	}
	text, err = execute(game.New(7), c)
	if err != nil || !strings.Contains(text, "暂无军事记录") {
		t.Fatal("empty history not explained")
	}
}
