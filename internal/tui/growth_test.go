package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
)

func TestGrowthPreviewShowsMultiplePeopleAndPreservesSmallScreenFeedback(t *testing.T) {
	s := chartState(t, [3]int{295, 1, 1}) // Capacity 1500, enough room for 1200 people.
	s.Tribes[0].Population, s.Tribes[0].Food = 1200, 100000
	m := colorModel(t, s, colorprofile.TrueColor)
	before := snapshot(s)
	for _, size := range [][2]int{{80, 24}, {80, 32}, {100, 32}, {120, 36}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		text := sideText(m)
		for _, want := range []string{"人口：1200/1500", "增长0/1000", "安全粮735", "预计+18人", "连续缺粮：0月"} {
			if !strings.Contains(text, want) {
				t.Fatalf("size %v lost %q: %s", size, want, text)
			}
		}
	}
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("growth preview issued people or spent stocks")
	}
}
