package tui

import (
	"fmt"

	"tribal-age/internal/game"
)

// View cells already have deterministic y/x order and contain all owned land.
func ownedCells(v game.View) []game.Cell {
	var owned []game.Cell
	for _, c := range v.Cells {
		if c.Tile.Owner == v.Self.ID {
			owned = append(owned, c)
		}
	}
	return owned
}

func (m *model) openTerritories() {
	m.dialog, m.territoryIndex = territoryDialog, 0
	for i, c := range ownedCells(m.playerView()) {
		if c.Point == m.cursor {
			m.territoryIndex = i
			return
		}
	}
}

func (m *model) territoryKey(key string) {
	cells := ownedCells(m.playerView())
	last := max(0, len(cells)-1)
	m.territoryIndex = min(last, max(0, m.territoryIndex))
	switch key {
	case "esc", "l":
		m.dialog = noDialog
	case "up", "w":
		m.territoryIndex = max(0, m.territoryIndex-1)
	case "down", "s":
		m.territoryIndex = min(last, m.territoryIndex+1)
	case "home":
		m.territoryIndex = 0
	case "enter":
		if len(cells) > 0 {
			m.cursor = cells[m.territoryIndex].Point
		}
		m.dialog = noDialog
	}
}

func territoryActivity(v game.View, f game.Forecast) map[game.Point]string {
	activity := make(map[game.Point]string)
	for _, p := range f.Production.Farms {
		activity[p] = fmt.Sprintf("预计产粮%d", game.FarmYield(v.Calendar.Season()))
	}
	for _, p := range f.Production.Forests {
		activity[p] = fmt.Sprintf("预计采木%d", game.ForestYield)
	}
	for _, p := range f.Production.Unstaffed {
		activity[p] = "人手不足，预计停产"
	}
	for _, c := range v.Cells {
		if reason := v.ProductionPause(c); reason != "" {
			activity[c.Point] = reason + "，暂停经营"
		}
	}
	return activity
}

func inactivePlot(c game.Cell, season game.Season) string {
	if c.Tile.Farm == game.FarmReady && game.FarmYield(season) == 0 {
		return "冬季休耕"
	}
	if c.Tile.Farm != game.FarmNone {
		return "建造中，本月无收成"
	}
	return "-"
}
