package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"tribal-age/internal/game"
)

func key(name string) tea.KeyPressMsg {
	codes := map[string]rune{"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight, "enter": tea.KeyEnter, "esc": tea.KeyEscape, "home": tea.KeyHome}
	if code, ok := codes[name]; ok {
		return tea.KeyPressMsg{Code: code}
	}
	if name == "ctrl+c" {
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	}
	return tea.KeyPressMsg{Code: []rune(name)[0], Text: name}
}

func press(m *model, names ...string) tea.Cmd {
	var cmd tea.Cmd
	for _, name := range names {
		_, cmd = m.Update(key(name))
	}
	return cmd
}

func snapshot(s *game.State) game.State {
	before := *s
	before.Events = append([]game.Event(nil), s.Events...)
	before.Reports = append([]game.MilitaryReport(nil), s.Reports...)
	for i := range before.Reports {
		before.Reports[i].Changes = append([]game.ArmyChange(nil), s.Reports[i].Changes...)
		before.Reports[i].Battles = append([]game.BattleReport(nil), s.Reports[i].Battles...)
		for j := range before.Reports[i].Battles {
			before.Reports[i].Battles[j].Armies = append([]game.BattleArmy(nil), s.Reports[i].Battles[j].Armies...)
		}
	}
	return before
}

func TestNavigationAndWindowEventsAreFree(t *testing.T) {
	s := game.New(42)
	m := newModel(s)
	if m.cursor != s.Tribes[0].Capital || m.Init() != nil {
		t.Fatal("game did not focus its capital")
	}
	before := snapshot(s)
	for _, direction := range []string{"up", "left", "down", "right"} {
		for i := 0; i < 50; i++ {
			press(m, direction)
		}
	}
	if m.cursor != (game.Point{X: game.Width - 1, Y: game.Height - 1}) {
		t.Fatal("cursor did not clamp at map edges")
	}
	press(m, "w", "a", "s", "d", "?", "esc", "z")
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Update(tea.KeyReleaseMsg{Code: 'b'})
	m.Update(nil)
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("navigation, help or terminal events changed the game")
	}
}

func TestKeyboardOpeningMatchesPublicEngineActions(t *testing.T) {
	s, oracle := game.New(42), game.New(42)
	m := newModel(s)
	p := s.Tribes[0].Capital
	press(m, "b")
	if err := oracle.Apply(0, game.Action{Kind: game.BuildFarm, Target: p}); err != nil {
		t.Fatal(err)
	}
	press(m, "up", "e")
	if err := oracle.Apply(0, game.Action{Kind: game.Explore, Target: game.Point{X: p.X, Y: p.Y - 1}}); err != nil {
		t.Fatal(err)
	}
	press(m, "enter", "y", "c")
	if err := oracle.EndMonth(game.Interactive); err != nil {
		t.Fatal(err)
	}
	if err := oracle.Apply(0, game.Action{Kind: game.Claim, Target: game.Point{X: p.X, Y: p.Y - 1}}); err != nil {
		t.Fatal(err)
	}
	for month := 0; month < 3; month++ {
		press(m, "enter", "y")
		if err := oracle.EndMonth(game.Interactive); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(s, oracle) || s.Tribes[0].Population != 31 {
		t.Fatal("keyboard interaction changed the game rules")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidActionsReportReasonWithoutCharges(t *testing.T) {
	s := game.New(42)
	m := newModel(s)
	press(m, "b")
	before := snapshot(s)
	press(m, "b")
	if !reflect.DeepEqual(before, *s) || !strings.Contains(m.View().Content, "已有在建或完工农田") {
		t.Fatal("invalid farm action charged resources or hid its reason")
	}
	press(m, "right", "c")
	if !reflect.DeepEqual(before, *s) || !strings.Contains(m.View().Content, "尚未探索") {
		t.Fatal("invalid claim changed the game")
	}
}

func TestMonthAndQuitRequireExplicitConfirmation(t *testing.T) {
	s := game.New(42)
	m := newModel(s)
	before := snapshot(s)
	press(m, "enter", "enter", "enter", "n", "enter", "esc")
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("accidental enter or cancelled confirmation ended a month")
	}
	press(m, "enter", "y", "y")
	if s.Calendar.Elapsed != 1 {
		t.Fatal("confirmation did not advance exactly one month")
	}
	before = snapshot(s)
	if cmd := press(m, "q"); cmd != nil {
		t.Fatal("quit did not ask for confirmation")
	}
	if !strings.Contains(m.View().Content, "存档") {
		t.Fatal("quit did not warn about lost progress")
	}
	if cmd := press(m, "esc"); cmd != nil {
		t.Fatal("cancelled quit exited")
	}
	press(m, "ctrl+c")
	cmd := press(m, "y")
	if cmd == nil {
		t.Fatal("confirmed quit did not exit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("confirmed quit returned the wrong command")
	}
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("quit changed the game")
	}
}

func TestTrainingAndProductionMenus(t *testing.T) {
	s := game.New(42)
	m := newModel(s)
	before := snapshot(s)
	press(m, "t", "esc", "p", "right", "right", "esc")
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("cancelled menus changed the game")
	}
	press(m, "t", "y")
	if s.Tribes[0].Soldiers != 10 || s.Tribes[0].AP != 1 || s.Tribes[0].Food != 55 {
		t.Fatal("training bypassed engine rules")
	}
	s.Tribes[0].AP = 0
	before = snapshot(s)
	press(m, "p", "right", "right", "right", "right")
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("production draft mutated the game")
	}
	press(m, "enter")
	if s.Tribes[0].ForestPriority != 4 || s.Tribes[0].AP != 0 || s.Calendar.Elapsed != 0 || s.Tribes[0].Food != 55 {
		t.Fatal("production setting cost AP or produced immediately")
	}
	press(m, "p", "0", "enter")
	if s.Tribes[0].ForestPriority != 0 {
		t.Fatal("automatic production was not restored")
	}
}

func TestSmallTerminalDoesNotAllowBlindActions(t *testing.T) {
	s := game.New(42)
	m := newModel(s)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	before := snapshot(s)
	press(m, "b", "enter", "y", "p", "right", "enter")
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("hidden controls in a small window changed the game")
	}
	press(m, "q")
	if !strings.Contains(m.View().Content, "存档") {
		t.Fatal("small-window exit hid the lost-progress warning")
	}
	if cmd := press(m, "y"); cmd == nil {
		t.Fatal("small terminal prevented exit")
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	press(m, "b")
	if s.Tribes[0].Wood != 10 {
		t.Fatal("resized terminal did not restore play")
	}
}

func TestExtinctionStaysReadableAndCannotAdvance(t *testing.T) {
	s := game.New(42)
	s.Tribes[0].Population, s.Tribes[0].Soldiers, s.Tribes[0].Food = 1, 1, 0
	m := newModel(s)
	press(m, "enter", "y", "enter", "y")
	if s.Tribes[0].Alive() || !strings.Contains(m.View().Content, "灭亡") {
		t.Fatal("game over was not shown")
	}
	before := snapshot(s)
	press(m, "enter", "y", "b", "right", "?")
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("game over revived or advanced the tribe")
	}
}

func TestMenuBoundsUnknownKeysAndMessageHistory(t *testing.T) {
	s := game.New(42)
	m := newModel(s)
	before := snapshot(s)
	press(m, "?", "?", "p", "-", "down", "left")
	for i := 0; i < game.Width*game.Height+2; i++ {
		press(m, "+")
	}
	if !strings.Contains(m.View().Content, "384格") {
		t.Fatal("production target did not clamp at map size")
	}
	press(m, "home", "up", "=", "-", "down", "z", "esc")
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("bounded or cancelled input changed the game")
	}
	press(m, "t", "z")
	if !strings.Contains(m.View().Content, "训练士兵") {
		t.Fatal("unknown confirmation key executed or cancelled training")
	}
	press(m, "n", "B")
	for i := 0; i < messageLimit+2; i++ {
		press(m, "b")
	}
	if len(m.messages) > messageLimit || !strings.Contains(m.View().Content, "已有在建或完工农田") || s.Tribes[0].Wood != 10 || s.Tribes[0].AP != 1 {
		t.Fatal("message history grew unbounded or repeated failures spent resources")
	}
}
