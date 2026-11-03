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

func FuzzKeyboardAndResizeInvariants(f *testing.F) {
	f.Add([]byte{6, 0, 4, 9, 10, 5, 9, 10})
	f.Add([]byte{8, 3, 3, 3, 3, 9, 9, 10, 7, 11, 13, 12})
	f.Add([]byte{255, 0, 64, 127, 128, 9, 10, 14, 10})
	f.Add([]byte{18, 2, 2, 0, 9, 18, 12, 14, 10})
	f.Add([]byte{131, 19, 18, 12, 19, 140, 0, 2, 9})
	f.Setenv("NO_COLOR", "")
	f.Setenv("TERM", "xterm-256color")
	profiles := []colorprofile.Profile{colorprofile.Unknown, colorprofile.NoTTY, colorprofile.ASCII, colorprofile.ANSI, colorprofile.ANSI256, colorprofile.TrueColor}
	keys := []string{"up", "left", "down", "right", "e", "c", "b", "t", "p", "enter", "y", "n", "esc", "?", "q", "+", "-", "0", "l", "v"}
	f.Fuzz(func(t *testing.T, data []byte) {
		s := game.New(42)
		m := newModel(s)
		for i, value := range data[:min(len(data), 128)] {
			if value&64 != 0 {
				m.Update(tea.WindowSizeMsg{Width: int(value), Height: int(value % 40)})
			}
			if value&128 != 0 {
				m.Update(tea.ColorProfileMsg{Profile: profiles[int(value)%len(profiles)]})
			}
			cmd := press(m, keys[int(value)%len(keys)])
			if err := s.Validate(); err != nil {
				t.Fatal(err)
			}
			if i%8 == 0 {
				before := snapshot(s)
				content := m.View().Content
				if !reflect.DeepEqual(before, *s) {
					t.Fatal("render changed game state")
				}
				if m.fits() && !strings.Contains(content, "部落纪元") {
					t.Fatal("valid terminal lost the game screen")
				}
				if content != "" {
					lines := strings.Split(content, "\n")
					if len(lines) > m.height {
						t.Fatal("height overflow")
					}
					for _, line := range lines {
						if ansi.StringWidth(line) > m.width {
							t.Fatal("width overflow")
						}
					}
				}
			}
			if cmd != nil {
				break
			} // A confirmed exit ends the event stream.
		}
	})
}
