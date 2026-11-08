package tui

import (
	"bytes"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"reflect"
	"strings"
	"testing"
	"tribal-age/internal/game"
)

func cutoffModel() *model {
	s := game.New(42)
	p := game.Point{X: 1, Y: 1}
	s.Tiles[p.Y][p.X] = game.Tile{Terrain: game.Grassland, Owner: game.Unowned}
	s.Tribes[0].Explored[p.Y][p.X] = true
	s.Tribes[0].Armies[0] = game.Army{People: 1, Position: p, Unsupplied: 1}
	return newModel(s)
}
func TestArmyDeathOpensReportAfterExactlyOneSavedMonth(t *testing.T) {
	m := cutoffModel()
	writes := 0
	var saved []byte
	m.save = func(s *game.State) error { writes++; var err error; saved, err = s.MarshalSave(); return err }
	press(m, "enter", "y")
	if m.dialog != reportDialog || writes != 1 || m.state.Calendar.Elapsed != 1 {
		t.Fatal("death did not open read-only report after one settlement/save")
	}
	text := m.View().Content
	for _, word := range []string{"军事战报", "1 → 0", "持续断供", "编制已移除"} {
		if !strings.Contains(text, word) {
			t.Fatalf("missing disappearance reason %s", word)
		}
	}
	press(m, "enter")
	press(m, "b") // An unrelated event must not push the summary out.
	if !strings.Contains(m.View().Content, "断供1") || !strings.Contains(m.View().Content, "R战报") {
		t.Fatal("important monthly summary was lost in production/activity log")
	}
	press(m, "m")
	if !strings.Contains(m.View().Content, "已覆灭") || !strings.Contains(m.View().Content, "持续断供") {
		t.Fatal("destroyed army looks never assembled")
	}
	press(m, "esc")
	restored, err := game.UnmarshalSave(saved)
	if err != nil {
		t.Fatal(err)
	}
	loaded := newModel(restored)
	press(loaded, "R")
	if !strings.Contains(loaded.View().Content, "持续断供") {
		t.Fatal("saved battle history could not be reopened")
	}
	before, err := m.state.MarshalSave()
	if err != nil {
		t.Fatal(err)
	}
	press(m, "r", "down", "pagedown", "home", "end", "left", "right", "enter")
	after, err := m.state.MarshalSave()
	if err != nil || !bytes.Equal(before, after) || writes != 1 {
		t.Fatal("browsing reports changed rules or wrote a save")
	}
}
func TestReportEmptyHistoryAndSmallScreenAreSafe(t *testing.T) {
	m := newModel(game.New(42))
	before := snapshot(m.state)
	press(m, "r")
	if !strings.Contains(m.View().Content, "暂无军事记录") {
		t.Fatal("empty history lacks explanation")
	}
	press(m, "enter")
	if m.dialog != noDialog {
		t.Fatal("return from report triggered month confirmation")
	}
	m = cutoffModel()
	press(m, "enter", "y")
	for _, size := range [][2]int{{80, 24}, {120, 36}, {160, 48}, {79, 23}, {0, 0}, {200, 70}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		before = snapshot(m.state)
		press(m, "r", "down", "up", "left", "right", "pagedown", "pageup", "end", "home")
		if !reflect.DeepEqual(before, *m.state) {
			t.Fatal("report navigation at resized terminal changed game")
		}
		text := m.View().Content
		for _, line := range strings.Split(text, "\n") {
			if ansi.StringWidth(line) > size[0] {
				t.Fatal("report overflowed width")
			}
		}
		if text != "" && len(strings.Split(text, "\n")) > size[1] {
			t.Fatal("report overflowed height")
		}
	}
}
