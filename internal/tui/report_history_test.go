package tui

import (
	"bytes"
	"strings"
	"testing"
	"tribal-age/internal/game"
)

func TestLegacyMessagesScrollWithoutLeakingAndNewReportsSwitchMonths(t *testing.T) {
	s := game.New(42)
	for i := 0; i < 5; i++ {
		if err := s.EndMonth(game.Simulation); err != nil {
			t.Fatal(err)
		}
	}
	s.ReportsSince = s.Calendar.Elapsed // Same boundary assigned when loading version 1.
	s.Events = append(s.Events, game.Event{Month: 0, Audience: 2, Text: "HIDDEN_ENEMY_TREASURY"})
	m := newModel(s)
	press(m, "r")
	_, lines := m.reportBody()
	if strings.Contains(strings.Join(lines, "\n"), "HIDDEN_ENEMY_TREASURY") || !strings.Contains(m.View().Content, "更早不补造战果") {
		t.Fatal("legacy fallback leaked hidden history or invented facts")
	}
	before, err := s.MarshalSave()
	if err != nil {
		t.Fatal(err)
	}
	press(m, "end")
	if m.reportOffset == 0 {
		t.Fatal("long original message history cannot scroll")
	}
	press(m, "pageup", "pagedown", "home")
	if m.reportOffset != 0 {
		t.Fatal("report home did not return to beginning")
	}
	after, err := s.MarshalSave()
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("legacy browsing changed state")
	}
	m = cutoffModel()
	press(m, "enter", "y", "enter", "m", "enter", "r")
	if !strings.Contains(m.View().Content, "本月未结算") {
		t.Fatal("current-month changes look durable")
	}
	press(m, "left")
	if !strings.Contains(m.View().Content, "持续断供") {
		t.Fatal("cannot revisit prior disappearance")
	}
	press(m, "right")
	if !strings.Contains(m.View().Content, "人员未死亡") {
		t.Fatal("reused army slot lost reformation distinction")
	}
}
func TestAutosaveFailureRemainsVisibleInsideFatalReport(t *testing.T) {
	m := cutoffModel()
	m.save = func(*game.State) error { return bytes.ErrTooLarge }
	press(m, "enter", "y")
	if m.dialog != reportDialog || !strings.Contains(m.View().Content, "自动存档失败") || !strings.Contains(m.View().Content, "持续断供") || m.state.Calendar.Elapsed != 1 {
		t.Fatal("automatic report concealed save failure or repeated settlement")
	}
}
