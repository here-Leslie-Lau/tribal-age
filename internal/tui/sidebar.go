package tui

import (
	"fmt"

	"tribal-age/internal/game"
)

func (m *model) sidebarGroups(v game.View, f game.Forecast, l layout) [][]string {
	t, p := v.Self, f.Production
	compact := !l.roomy
	name, policy := t.Name, "生产：自动（农田优先）"
	if !t.Alive() {
		name += " [灭亡]"
	}
	if m.dialog == productionDialog {
		name += " [草稿]"
	}
	if t.ForestPriority > 0 {
		policy = fmt.Sprintf("生产：优先采木%d格", t.ForestPriority)
	}
	growth := "增长：本月可积累"
	if f.Growth.Reason != "" {
		growth = "增长：" + f.Growth.Reason
	}
	hunger := fmt.Sprintf("连续缺粮：%d月", t.HungerMonths)
	if f.Shortage > 0 {
		hunger = fmt.Sprintf("缺粮预警 %d粮 / 连缺%d月", f.Shortage, t.HungerMonths)
	}
	population := fmt.Sprintf("人口：%d/%d", t.Population, f.Growth.Capacity)
	identity := []string{name, population, fmt.Sprintf("士兵%d / 平民%d", t.Soldiers, t.Population-t.Soldiers)}
	if compact {
		identity = []string{fmt.Sprintf("%s 兵%d 民%d", name, t.Soldiers, t.Population-t.Soldiers), population}
	}
	return [][]string{
		identity,
		m.mapView.factionLines(v, !compact, l.rightWidth),
		{fmt.Sprintf("粮食：%d (上月%+d)", t.Food, t.LastChange.Food), fmt.Sprintf("收支 %+d / 余粮 %d", f.NetFood, f.FoodAfter),
			fmt.Sprintf("木材：%d (上月%+d)", t.Wood, t.LastChange.Wood), fmt.Sprintf("本月预计 %+d", p.Wood)},
		{fmt.Sprintf("人手%d格：田%d 林%d", p.Slots, len(p.Farms), len(p.Forests)), policy,
			fmt.Sprintf("增长%d/%d / 安全粮%d", t.GrowthProgress, game.GrowthMonths, f.Growth.Reserve), hunger, growth},
	}
}

// Spend spare rows on group gaps, not on clipping resources or growth reasons.
func (m *model) sidebarLines(v game.View, f game.Forecast, l layout) []string {
	groups := m.sidebarGroups(v, f, l)
	rows := 0
	for i, group := range groups {
		groups[i] = wrapLines(group, l.rightWidth)
		rows += len(groups[i])
	}
	gaps := 0
	if l.roomy {
		gaps = min(len(groups)-1, max(0, l.rows+2-rows))
	}
	var lines []string
	for i, group := range groups {
		if i > 0 && i <= gaps {
			lines = append(lines, "")
		}
		lines = append(lines, group...)
	}
	return lines
}
