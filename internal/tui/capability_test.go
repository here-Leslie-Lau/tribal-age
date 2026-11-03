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

func TestColorCapabilityChangesKeepModeCursorAndGame(t *testing.T) {
	m := colorModel(t, productionState(), colorprofile.TrueColor)
	press(m, "v")
	before := snapshot(m.state)
	first := m.View().Content
	cursor := m.cursor
	m.Update(tea.ColorProfileMsg{Profile: colorprofile.ASCII})
	plain := m.View().Content
	if strings.Contains(plain, "\x1b[") || !strings.Contains(plain, "地形视图(无色)") || !strings.Contains(plain, "[.@]") || !strings.Contains(plain, "fP") {
		t.Fatal("capability downgrade lost map mode, ownership or focus")
	}
	m.Update(tea.ColorProfileMsg{Profile: colorprofile.TrueColor})
	if m.View().Content != first || m.cursor != cursor || !reflect.DeepEqual(before, *m.state) {
		t.Fatal("capability change was not reversible and free")
	}
}

func TestFogAndFactionBadgesUseTheSamePaletteInBothModes(t *testing.T) {
	m := colorModel(t, ownershipState(), colorprofile.TrueColor)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	m.cursor = game.Point{}
	fog := background(mapCell(m, m.cursor))
	for mode := 0; mode < 2; mode++ {
		if background(mapCell(m, m.cursor)) != fog {
			t.Fatal("fog changed with hidden map mode")
		}
		for _, token := range territoryTones {
			badge := m.mapView.paint("  ", token, false)
			if !strings.Contains(m.View().Content, badge) || ansi.StringWidth(badge) != 2 {
				t.Fatal("faction badge did not match map palette")
			}
		}
		press(m, "v")
	}
}
