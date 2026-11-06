package tui

import (
	tea "charm.land/bubbletea/v2"
	"tribal-age/internal/game"
)

func (m *model) confirm(key string) tea.Cmd {
	if cancelsConfirmation(key, m.dialog) {
		m.dialog = noDialog
		return nil
	}
	if key != "y" {
		return nil
	}
	chosen := m.dialog
	m.dialog = noDialog
	switch chosen {
	case quitDialog:
		return tea.Quit
	case endDialog:
		m.endMonth()
	case trainingDialog:
		m.apply(game.Train)
	}
	return nil
}

func (m *model) editProduction(key string) {
	switch key {
	case "esc":
		m.dialog = noDialog
	case "right", "up", "+", "=":
		m.workTarget = min(game.Width*game.Height, m.workTarget+1)
	case "left", "down", "-":
		m.workTarget = max(0, m.workTarget-1)
	case "0", "home":
		m.workTarget = 0
	case "enter":
		from := len(m.state.Events)
		m.record(from, m.state.SetForestPriority(0, m.workTarget))
		m.dialog = noDialog
	}
}

func cancelsConfirmation(key string, kind dialog) bool {
	switch key {
	case "esc", "n", "enter":
		return true
	case "?":
		return kind == helpDialog
	}
	return false
}
