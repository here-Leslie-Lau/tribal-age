package tui

import (
	"errors"
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"tribal-age/internal/game"
	"tribal-age/internal/savegame"
)

func TestContinueRestoresOnlyPreviouslyVisibleMessages(t *testing.T) {
	store := saveStore(t)
	s := game.New(42)
	if err := savegame.EndMonth(s, store.Save); err != nil {
		t.Fatal(err)
	}
	s.Events = append(s.Events, game.Event{Month: 0, Audience: 1, Text: "我方可见的月末战报"}, game.Event{Month: 0, Audience: 2, Text: "不应泄露的敌方库存"})
	if err := store.Save(s); err != nil {
		t.Fatal(err)
	}
	m, err := newSavedModel(7, store)
	if err != nil {
		t.Fatal(err)
	}
	press(m, "enter")
	if !strings.Contains(m.View().Content, "我方可见的月末战报") || strings.Contains(m.View().Content, "不应泄露的敌方库存") {
		t.Fatal("continue lost visible reports or exposed private events")
	}
}
func TestStartupSizesNavigationAndSafeCancel(t *testing.T) {
	store := saveStore(t)
	if err := savegame.EndMonth(game.New(42), store.Save); err != nil {
		t.Fatal(err)
	}
	m, err := newSavedModel(7, store)
	if err != nil {
		t.Fatal(err)
	}
	press(m, "up", "z")
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	press(m, "enter", "down", "y")
	if m.startup == nil {
		t.Fatal("undersized startup accepted hidden choice")
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	press(m, "down", "enter", "z", "n", "down")
	if cmd := press(m, "enter"); cmd == nil {
		t.Fatal("startup exit failed")
	}
	if _, err := newSavedModel(1, nil); err == nil {
		t.Fatal("nil save session accepted")
	}
	store.Close()
	if _, err := newSavedModel(1, store); err == nil {
		t.Fatal("closed save session accepted")
	}
}
func TestSavedRunnerReportsIOErrorsWithoutWriting(t *testing.T) {
	store := saveStore(t)
	if err := RunSaved(42, store, nil, io.Discard); err == nil {
		t.Fatal("nil input accepted")
	}
	if err := RunSaved(42, store, strings.NewReader(""), nil); err == nil {
		t.Fatal("nil output accepted")
	}
	if err := RunSaved(42, nil, strings.NewReader(""), io.Discard); err == nil {
		t.Fatal("nil session accepted")
	}
	if err := RunSaved(42, store, errorReader{err: io.ErrUnexpectedEOF}, io.Discard); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal("lost runtime input error")
	}
	if _, err := store.Load(); err == nil {
		t.Fatal("terminal failure created save")
	}
}
