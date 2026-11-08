package game

import "fmt"

func validateSavedReports(s *State) error {
	if !reportRange(s.ReportsSince, 0, s.Calendar.Elapsed) {
		return fmt.Errorf("战报起始月份无效")
	}
	lastMonth := -1
	var owners uint8
	for _, r := range s.Reports {
		if r.Month < lastMonth {
			return fmt.Errorf("战报月份次序无效")
		}
		if r.Month != lastMonth {
			owners = 0
			lastMonth = r.Month
		}
		if !reportRange(r.Owner, 0, TribeCount-1) || owners&(1<<r.Owner) != 0 {
			return fmt.Errorf("战报部落或重复月份无效")
		}
		owners |= 1 << r.Owner
		if err := validateReport(r, s.ReportsSince, s.Calendar.Elapsed); err != nil {
			return err
		}
	}
	return nil
}
func validateReport(r MilitaryReport, since, elapsed int) error {
	if !reportRange(r.Month, since, elapsed) || len(r.Battles)+len(r.Changes) == 0 {
		return fmt.Errorf("战报日期或内容无效")
	}
	combat, err := reportBattleLosses(r)
	if err != nil {
		return err
	}
	for _, c := range r.Changes {
		if err := validateArmyChange(c); err != nil {
			return err
		}
		if c.Cause == CombatLoss {
			combat[c.Slot] -= c.Before - c.After
		}
	}
	for _, n := range combat {
		if n != 0 {
			return fmt.Errorf("战斗阵亡与部队变化不一致")
		}
	}
	return nil
}
func validateArmyChange(c ArmyChange) error {
	if !reportRange(c.Slot, ReserveSlot, MaxArmies-1) || !c.Position.Valid() || c.Cause > DefeatDisbanded {
		return fmt.Errorf("部队变化身份、位置或原因无效")
	}
	if c.Slot == ReserveSlot && c.Cause != FamineLoss {
		return fmt.Errorf("未编组士兵变化原因无效")
	}
	return validateChangeNumbers(c)
}
func validateBattleReport(b BattleReport, owner int) error {
	if len(b.Armies) < 2 || len(b.Armies) > TribeCount*MaxArmies {
		return fmt.Errorf("交锋参战记录不完整")
	}
	var seen [TribeCount][MaxArmies]bool
	own, enemy := false, false
	for _, a := range b.Armies {
		if err := validateBattleArmy(a); err != nil {
			return err
		}
		if seen[a.Owner][a.Slot] {
			return fmt.Errorf("交锋重复计算同一部队")
		}
		seen[a.Owner][a.Slot] = true
		if a.Owner == owner {
			own = true
		} else {
			enemy = true
		}
	}
	if !own || !enemy {
		return fmt.Errorf("交锋缺少己方或对手")
	}
	return nil
}
func validateBattleArmy(a BattleArmy) error {
	if !reportRange(a.Owner, 0, TribeCount-1) || !reportRange(a.Slot, 0, MaxArmies-1) {
		return fmt.Errorf("参战部队身份无效")
	}
	if !reportRange(a.Before, 1, MaxArmyPeople) || !reportRange(a.Lost, 0, a.Before) {
		return fmt.Errorf("参战部队阵亡人数无效")
	}
	if !validBattleRoute(a) {
		return fmt.Errorf("参战部队路线、补给或防御无效")
	}
	return nil
}

func reportRange(n, low, high int) bool { return n >= low && n <= high }
func reportBattleLosses(r MilitaryReport) ([MaxArmies]int, error) {
	var losses [MaxArmies]int
	for _, b := range r.Battles {
		if err := validateBattleReport(b, r.Owner); err != nil {
			return losses, err
		}
		for _, a := range b.Armies {
			if a.Owner == r.Owner {
				losses[a.Slot] += a.Lost
			}
		}
	}
	return losses, nil
}
func validateChangeNumbers(c ArmyChange) error {
	limit := MaxArmyPeople
	if c.Slot == ReserveSlot {
		limit = saveNumberLimit
	}
	if !reportRange(c.Before, 0, limit) || !reportRange(c.After, 0, limit) || c.Before == c.After {
		return fmt.Errorf("部队变化人数无效")
	}
	if c.Cause != Reorganized && c.After >= c.Before {
		return fmt.Errorf("减员记录不得增加人数")
	}
	if c.Cause == DefeatDisbanded && c.After != 0 {
		return fmt.Errorf("战败解散后不得保留编制")
	}
	return nil
}
func validBattleRoute(a BattleArmy) bool {
	return a.From.Valid() && a.Target.Valid() && manhattan(a.From, a.Target) <= 1 && a.Supply <= CutOff && (!a.ForestDefense || a.From == a.Target)
}
