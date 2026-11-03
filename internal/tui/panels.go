package tui

import (
	"fmt"

	"tribal-age/internal/game"
)

const confirmHint = "[Y]确认 [N/Esc/Enter]取消（默认取消）"

func (m *model) panelLines(v game.View, f game.Forecast, width, rows int) []string {
	switch m.dialog {
	case endDialog:
		return []string{"确认结束本月？", fmt.Sprintf("还剩%d行动点；电脑随后行动，并结算本月生产、粮耗和人口。", v.Self.AP),
			fmt.Sprintf("粮食收支%+d，预计余粮%d，预计缺粮%d。", f.NetFood, f.FoodAfter, f.Shortage), confirmHint}
	case quitDialog:
		return []string{"确认退出游戏？", "尚不支持存档，退出或关闭终端后，本局进度会丢失。", confirmHint}
	case trainingDialog:
		return []string{"训练士兵？", fmt.Sprintf("%d名平民转为士兵，总人口不变；花费%d粮、%d行动点。", game.TrainingSize, game.TrainingFood, game.ActionAP),
			"士兵不生产，还会增加口粮需求；实际条件由引擎检查。", confirmHint}
	case productionDialog:
		return productionPanel(v, f)
	case helpDialog:
		return []string{"帮助：方向键/WASD移动光标，不花行动点。", "E探索 C占领 B建田；T训练 P生产 L领地一览。",
			"Enter结束月份、Q退出都需Y确认；只有结束月份推进时间。", "地图每格：地形+归属，>标记选中位置。Esc/?返回。"}
	}
	heading := "最近消息："
	if !v.Self.Alive() {
		heading = "部落灭亡，本局结束；可查看地图或Q退出。"
	}
	messages := wrapLines(m.messages, width)
	from := max(0, len(messages)-(rows-1))
	return append([]string{heading}, messages[from:]...)
}

func productionPanel(v game.View, f game.Forecast) []string {
	p := f.Production
	note := fmt.Sprintf("实际安排：%d田、%d林；其余人手自动安排。", len(p.Farms), len(p.Forests))
	if len(p.Forests) < v.Self.ForestPriority {
		note = fmt.Sprintf("森林或人手不足，目标未满足：实际%d田、%d林。", len(p.Farms), len(p.Forests))
	}
	return []string{fmt.Sprintf("生产安排（草稿）：优先采木%d格，0表示自动。", v.Self.ForestPriority),
		"方向键或 +/- 调数量；0恢复自动；Enter应用，Esc取消。", note,
		fmt.Sprintf("预计产粮%d、产木%d；收支%+d，余粮%d，缺粮%d。", p.Food, p.Wood, f.NetFood, f.FoodAfter, f.Shortage)}
}
