package tui

import (
	"bytes"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func FuzzMilitaryReportNavigation(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6, 7})
	f.Add([]byte{255, 129, 140, 201, 64, 0, 7})
	keys := []string{"up", "down", "left", "right", "pageup", "pagedown", "home", "end"}
	f.Fuzz(func(t *testing.T, data []byte) {
		m := battleAndCutoffModel()
		press(m, "enter", "y")
		before, err := m.state.MarshalSave()
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range data[:min(len(data), 64)] {
			if value&128 != 0 {
				m.Update(tea.WindowSizeMsg{Width: int(value), Height: int(value % 49)})
			}
			press(m, keys[int(value)%len(keys)])
			content := m.View().Content
			for _, line := range strings.Split(content, "\n") {
				if ansi.StringWidth(line) > m.width {
					t.Fatal("report width overflow")
				}
			}
			if content != "" && len(strings.Split(content, "\n")) > m.height {
				t.Fatal("report height overflow")
			}
		}
		after, err := m.state.MarshalSave()
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("report navigation/rendering changed saved facts")
		}
	})
}
