package tui

import (
	"fmt"
	"strings"

	"tribal-age/internal/game"
)

func (m *model) armyGameKey(key string) bool {
	switch key {
	case "m":
		m.dialog = armyDialog
		m.resetArmyDraft()
	case "1", "2", "3":
		m.armySlot = int(key[0] - '1')
		a := m.state.Tribes[0].Armies[m.armySlot]
		if a.People > 0 {
			m.cursor = a.Position
		}
	case "f":
		m.issueArmyOrder(game.ArmyOrder{Kind: game.ArmyApproachKind(m.playerView(), m.cursor), Target: m.cursor})
	case "o":
		m.issueArmyOrder(game.ArmyOrder{Kind: game.Occupy})
	case "h":
		m.issueArmyOrder(game.ArmyOrder{Kind: game.Hold})
	default:
		return false
	}
	return true
}
func (m *model) issueArmyOrder(order game.ArmyOrder) {
	from := len(m.state.Events)
	m.record(from, m.state.SetArmyOrder(0, m.armySlot, order))
}
func (m *model) resetArmyDraft() {
	t := m.state.Tribes[0]
	m.armyPeople = t.Armies[m.armySlot].People
	if m.armyPeople == 0 {
		m.armyPeople = min(game.TrainingSize, t.ReserveSoldiers())
	}
}
func (m *model) armyKey(key string) {
	t := m.state.Tribes[0]
	limit := min(game.MaxArmyPeople, t.ReserveSoldiers()+t.Armies[m.armySlot].People)
	switch key {
	case "esc", "m":
		m.dialog = noDialog
	case "1", "2", "3":
		m.armySlot = int(key[0] - '1')
		m.resetArmyDraft()
	case "+", "=", "right", "up":
		m.armyPeople = min(limit, m.armyPeople+1)
	case "-", "left", "down":
		m.armyPeople = max(0, m.armyPeople-1)
	case "0", "home":
		m.armyPeople = 0
	case "enter":
		from := len(m.state.Events)
		m.record(from, m.state.SetArmySize(0, m.armySlot, m.armyPeople))
		m.dialog = noDialog
	}
}
func (m *model) armyScreen() string {
	l := screenLayout(m.width, m.height, m.cursor)
	v := m.playerView()
	t := v.Self
	lines := []string{fmt.Sprintf("部落纪元 | 军队管理 | %s | 行动点%d/%d", v.Calendar, t.AP, game.MonthlyAP), "",
		fmt.Sprintf("士兵%d（属于总人口%d） / 未编组%d；最多%d支军队。", t.Soldiers, t.Population, t.ReserveSoldiers(), game.MaxArmies), ""}
	for slot, a := range t.Armies {
		marker := "  "
		if slot == m.armySlot {
			marker = "> "
		}
		text := fmt.Sprintf("%s军队%c：%s", marker, 'A'+slot, m.armyEndNote(slot))
		if a.People > 0 {
			text = fmt.Sprintf("%s军队%c：%d人 (%d,%d) 补给%s / %s", marker, 'A'+slot, a.People, a.Position.X, a.Position.Y, visibleArmySupply(v, slot), armyOrderText(a.Order))
		}
		lines = append(lines, text)
	}
	lines = append(lines, "", fmt.Sprintf("草稿：军队%c调整为%d人；0表示解编。", 'A'+m.armySlot, m.armyPeople),
		"1/2/3选军 | +/-或方向键调人数 | 0解编 | Enter应用 | Esc取消",
		"调整人数需在安全首都办理，变化花1行动点；相同人数不扣点。", "",
		"返回地图：1/2/3定位军队，移动光标选相邻目标，F行军/进攻。",
		"O接收脚下敌地，H撤销军令并驻守；每军每月只执行一条。",
		"所有军令月底执行；经济预览不包含交战、断供与占领的结果。", "",
		"首版AI负责守备和反击，不主动跨大陆远征。")
	return l.frame(lines)
}
func visibleArmySupply(v game.View, slot int) game.SupplyLevel {
	for _, a := range v.Armies {
		if a.Owner == v.Self.ID && a.Slot == slot {
			return a.Supply
		}
	}
	return game.CutOff
}
func armyOrderText(o game.ArmyOrder) string {
	if o.Kind == game.March || o.Kind == game.Attack {
		return fmt.Sprintf("%s(%d,%d)", o.Kind, o.Target.X, o.Target.Y)
	}
	return o.Kind.String()
}
func (m *model) armyHint(v game.View) string {
	a := v.Self.Armies[m.armySlot]
	if estimate := game.AttackEstimate(v, m.armySlot); estimate.Available {
		return fmt.Sprintf("军%c %s 补给%s | 估损我%d敌%d（未计其他军令）", 'A'+m.armySlot, armyOrderText(a.Order), visibleArmySupply(v, m.armySlot), estimate.OwnLoss, estimate.EnemyLoss)
	}
	return fmt.Sprintf("军%c %d人 补给%s / %s | F目标 O占领 H驻守（月底执行）", 'A'+m.armySlot, a.People, visibleArmySupply(v, m.armySlot), armyOrderText(a.Order))
}
func visibleArmyText(v game.View, p game.Point) string {
	var parts []string
	for _, a := range v.Armies {
		if a.Position == p {
			parts = append(parts, fmt.Sprintf("%c%d:%d人", 'A'+a.Slot, a.Owner+1, a.People))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " / 军队" + strings.Join(parts, ",")
}
func armyMapGlyph(v game.View, p game.Point) string {
	for _, a := range v.Armies {
		if a.Position == p {
			return fmt.Sprintf("%c%d", 'A'+a.Slot, a.Owner+1)
		}
	}
	return ""
}
