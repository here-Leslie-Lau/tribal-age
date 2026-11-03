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

func TestVSwitchesMapMeaningWithoutChangingGameOrSelection(t *testing.T) {
	m := colorModel(t, productionState(), colorprofile.TrueColor)
	p := m.cursor
	before := snapshot(m.state)
	first := m.View().Content
	if !strings.Contains(first, "领土视图") {
		t.Fatal("map did not default to territorial ownership")
	}
	press(m, "V")
	if !strings.Contains(m.View().Content, "地形视图") || m.View().Content == first {
		t.Fatal("V did not switch map meaning")
	}
	grass := mapCell(m, game.Point{X: p.X + 1, Y: p.Y})
	forest := mapCell(m, game.Point{X: p.X + 2, Y: p.Y})
	if background(grass) == "" || background(grass) == background(forest) {
		t.Fatal("terrain mode did not distinguish grass and forest")
	}
	m.Update(tea.KeyReleaseMsg{Code: 'v'})
	press(m, "l", "v", "enter")
	if !strings.Contains(m.View().Content, "地形视图") {
		t.Fatal("menu swallowed map preference or V changed a modal")
	}
	press(m, "v")
	if m.View().Content != first || m.cursor != p || !reflect.DeepEqual(before, *m.state) {
		t.Fatal("view switching was not reversible and free")
	}
}

func TestColoredMapFitsSizesAndDoesNotPaintSidebar(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {80, 32}, {100, 24}, {100, 32}, {120, 36}, {160, 48}} {
		m := colorModel(t, ownershipState(), colorprofile.TrueColor)
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for mode := 0; mode < 2; mode++ {
			content := m.View().Content
			if len(strings.Split(content, "\n")) > size[1] {
				t.Fatal("colored screen exceeded height")
			}
			for _, line := range strings.Split(content, "\n") {
				if ansi.StringWidth(line) > size[0] {
					t.Fatal("colored screen exceeded width")
				}
			}
			l := screenLayout(size[0], size[1], m.cursor)
			start := 4
			if l.roomy {
				start += 2
			}
			row := strings.Split(content, "\n")[start]
			last := strings.LastIndex(row, "\x1b[")
			if last < 0 || !strings.HasPrefix(row[last:], ansi.ResetStyle) {
				t.Fatal("map colors were not reset before other panels")
			}
			for _, label := range []string{"人口：31/40", "粮食：30", "木材：20", "安全粮24", "增长：", "Q退出", "V切换"} {
				if !strings.Contains(ansi.Strip(content), label) {
					t.Fatalf("screen %v hid %q", size, label)
				}
			}
			press(m, "v")
		}
	}
}
