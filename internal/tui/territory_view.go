package tui

import (
	"fmt"
	"strings"

	"tribal-age/internal/game"
)

func (m *model) territoryScreen() string {
	l := screenLayout(m.width, m.height, m.cursor)
	v := m.playerView()
	f := game.ForecastFor(v)
	cells := ownedCells(v)
	rows := l.height - 11
	index := min(max(0, m.territoryIndex), max(0, len(cells)-1))
	top := min(max(0, index-rows/2), max(0, len(cells)-rows))
	ready, building := farmCounts(cells)
	lines := []string{fmt.Sprintf("部落纪元 | %s领地一览 | %s", v.Self.Name, v.Calendar), "",
		fmt.Sprintf("  领地%d格  /  完工农田%d  /  在建农田%d", len(cells), ready, building),
		fmt.Sprintf("  人手可经营%d格；本月预计：田%d格、林%d格。", f.Production.Slots, len(f.Production.Farms), len(f.Production.Forests)), "",
		"    " + fitRow("位置", 12) + fitRow("地形", 8) + fitRow("农田情况", 24) + "本月预计", strings.Repeat("-", l.width)}
	activity := territoryActivity(v, f)
	for i := 0; i < rows; i++ {
		line := ""
		if top+i < len(cells) {
			line = territoryRow(cells[top+i], v, index == top+i, activity)
		}
		if len(cells) == 0 && i == 0 {
			line = "  暂无领地。"
		}
		lines = append(lines, line)
	}
	page := "暂无领地"
	if len(cells) > 0 {
		page = fmt.Sprintf("显示第%d-%d格，共%d格", top+1, min(len(cells), top+rows), len(cells))
	}
	lines = append(lines, "", "农田进度为当前进度；预计收成包含本月底完工的农田。", page, "上下键选择 | Enter定位到地图 | Esc/L返回（不花行动点）")
	return l.frame(lines)
}

func farmCounts(cells []game.Cell) (ready, building int) {
	for _, c := range cells {
		switch c.Tile.Farm {
		case game.FarmNone:
		case game.FarmReady:
			ready++
		default:
			building++
		}
	}
	return ready, building
}

func territoryRow(c game.Cell, v game.View, selected bool, activity map[game.Point]string) string {
	marker := "  "
	if selected {
		marker = "> "
	}
	position := fmt.Sprintf("(%d,%d)", c.Point.X, c.Point.Y)
	if c.Point == v.Self.Capital {
		position += "@"
	}
	status, ok := activity[c.Point]
	if !ok {
		status = inactivePlot(c, v.Calendar.Season())
	}
	return "  " + marker + fitRow(position, 12) + fitRow(c.Tile.Terrain.String(), 8) + fitRow(farmInfo(c.Tile.Farm), 24) + status
}
