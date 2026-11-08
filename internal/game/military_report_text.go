package game

import "fmt"

func armyLabel(slot int) string {
	if slot == ReserveSlot {
		return "未编组士兵"
	}
	return fmt.Sprintf("军队%c", 'A'+slot)
}
func (r MilitaryReport) Summary() string {
	own, enemy, supply, famine := 0, 0, 0, 0
	enemyNote := ""
	for _, b := range r.Battles {
		if b.Partial {
			enemyNote = "（已知）"
		}
		a, e := b.Losses(r.Owner)
		own += a
		enemy += e
	}
	for _, c := range r.Changes {
		switch c.Cause {
		case SupplyLoss:
			supply += c.Before - c.After
		case FamineLoss:
			famine += c.Before - c.After
		}
	}
	text := fmt.Sprintf("交锋%d 我亡%d 敌亡%d%s 断供%d 饥荒%d", len(r.Battles), own, enemy, enemyNote, supply, famine)
	if r.Destroyed() {
		text += "；部队覆灭/解散"
	}
	return text
}
func (b BattleReport) ratio(owner int) string {
	own, enemy := b.Losses(owner)
	if own == 0 {
		return fmt.Sprintf("我军无阵亡；敌军阵亡%d人。", enemy)
	}
	return fmt.Sprintf("阵亡比（我:敌）%d:%d", own, enemy)
}
func (a BattleArmy) line(owner int) string {
	side := "敌方"
	if a.Owner == owner {
		side = "我方"
	}
	outcome := "驻守原位"
	if a.From != a.Target {
		outcome = "未推进"
	}
	if a.Lost == a.Before {
		outcome = "覆灭"
	}
	defense := "无森林加成"
	if a.ForestDefense {
		defense = "森林驻守防御"
	}
	return fmt.Sprintf("%s 部落%d%s (%d,%d)→(%d,%d)：%d → %d，阵亡%d；补给%s，%s；%s。", side, a.Owner+1, armyLabel(a.Slot), a.From.X, a.From.Y, a.Target.X, a.Target.Y, a.Before, a.Before-a.Lost, a.Lost, a.Supply, defense, outcome)
}
func (c ArmyChange) Line() string {
	ending := ""
	if c.Removed() {
		ending = "；编制已移除"
	}
	return fmt.Sprintf("%s (%d,%d)：%d → %d，%s%s。", armyLabel(c.Slot), c.Position.X, c.Position.Y, c.Before, c.After, c.Cause, ending)
}

// Lines describe stored facts without requiring current visibility or live troops.
func (r MilitaryReport) Lines() []string {
	date := Calendar{Elapsed: r.Month}
	lines := []string{fmt.Sprintf("第%d年%d月 · 军事月报", date.Year(), date.Month()), r.Summary(), ""}
	if r.Destroyed() {
		lines = append(lines, "重要：部队去向")
	}
	for _, c := range r.Changes {
		if c.Removed() {
			lines = append(lines, c.Line())
		}
	}
	for i, b := range r.Battles {
		lines = append(lines, fmt.Sprintf("交锋%d：", i+1))
		for _, a := range b.Armies {
			lines = append(lines, a.line(r.Owner))
		}
		lines = append(lines, b.ratio(r.Owner), "敌方数字为参战部队本轮总阵亡，不是我军独占击杀数。")
		if b.Partial {
			lines = append(lines, "情报不完整：仅列本部落和直接交手部队，其他接敌不披露。")
		}
		lines = append(lines, "")
	}
	if len(r.Changes) > 0 {
		lines = append(lines, "部队变化（实际发生顺序，战斗项与上方同一批损失，不重复计数）：")
		for _, c := range r.Changes {
			lines = append(lines, c.Line())
		}
	}
	return lines
}
