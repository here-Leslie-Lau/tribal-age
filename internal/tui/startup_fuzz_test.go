package tui

import (
	tea "charm.land/bubbletea/v2"
	"errors"
	"github.com/charmbracelet/x/ansi"
	"reflect"
	"strings"
	"testing"
	"tribal-age/internal/game"
)

func FuzzStartupAndMonthSaveBoundaries(f *testing.F) {
	f.Add([]byte{0, 5, 5, 6, 6, 5, 6, 9, 6})
	f.Add([]byte{1, 2, 5, 7, 5, 6, 5, 6})
	f.Add([]byte{0, 2, 5, 6, 5, 6, 1, 3, 5, 6})
	f.Fuzz(func(t *testing.T, data []byte) {
		m := newModel(game.New(7))
		menu := &startupMenu{seed: 7, saved: game.New(42)}
		if len(data) > 0 && data[0]&1 != 0 {
			menu.saved = nil
			menu.problem = errors.New("存档损坏")
		}
		m.startup = menu
		writes := 0
		m.save = func(s *game.State) error {
			writes++
			b, err := s.MarshalSave()
			if err != nil {
				return err
			}
			loaded, err := game.UnmarshalSave(b)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(loaded, s) {
				t.Fatal("autosave lost rule state")
			}
			return nil
		}
		keys := []string{"up", "left", "down", "right", "b", "enter", "y", "n", "esc", "q", "m", "f", "o", "p", "+"}
		for _, value := range data[:min(len(data), 64)] {
			if value&128 != 0 {
				m.Update(tea.WindowSizeMsg{Width: int(value), Height: int(value % 50)})
			}
			k := keys[int(value)%len(keys)]
			expected := m.startup == nil && m.dialog == endDialog && k == "y" && m.fits() && m.state.Tribes[0].Alive()
			beforeWrites := writes
			cmd := press(m, k)
			if expected {
				if writes != beforeWrites+1 {
					t.Fatal("confirmed month did not save once")
				}
			} else if writes != beforeWrites {
				t.Fatal("startup, action, cancel or exit wrote archive")
			}
			if err := m.state.Validate(); err != nil {
				t.Fatal(err)
			}
			before := snapshot(m.state)
			text := m.View().Content
			if !reflect.DeepEqual(before, *m.state) {
				t.Fatal("menu rendering changed game")
			}
			if m.fits() && !strings.Contains(text, "部落纪元") {
				t.Fatal("lost startup/game heading")
			}
			for _, line := range strings.Split(text, "\n") {
				if ansi.StringWidth(line) > m.width {
					t.Fatal("startup width overflow")
				}
			}
			if text != "" && len(strings.Split(text, "\n")) > m.height {
				t.Fatal("startup height overflow")
			}
			if cmd != nil {
				break
			}
		}
	})
}
