package game

// A report belongs to one tribe. It stores facts captured before armies vanish,
// never references to live armies, hidden orders, inventories or distant fights.
type MilitaryReport struct {
	Owner, Month int
	Battles      []BattleReport
	Changes      []ArmyChange
}

type BattleReport struct {
	Armies  []BattleArmy
	Partial bool // Other contacts exist, but their troops and locations are not disclosed.
}

type BattleArmy struct {
	Owner, Slot   int
	Before, Lost  int
	From, Target  Point
	Supply        SupplyLevel
	ForestDefense bool
}

type ArmyChangeCause uint8

const (
	CombatLoss ArmyChangeCause = iota
	SupplyLoss
	FamineLoss
	Reorganized
	DefeatDisbanded
)
const ReserveSlot = -1

type ArmyChange struct {
	Slot          int
	Position      Point
	Before, After int
	Cause         ArmyChangeCause
}

func (c ArmyChangeCause) String() string {
	switch c {
	case CombatLoss:
		return "战斗阵亡"
	case SupplyLoss:
		return "持续断供"
	case FamineLoss:
		return "饥荒减员"
	case Reorganized:
		return "编组调整（人员未死亡）"
	case DefeatDisbanded:
		return "战败解散（未计为战死）"
	}
	return "未知原因"
}
func (c ArmyChange) Removed() bool {
	return c.Slot != ReserveSlot && c.Before > 0 && c.After == 0 && c.Cause != Reorganized
}
func (r MilitaryReport) Destroyed() bool {
	for _, c := range r.Changes {
		if c.Removed() {
			return true
		}
	}
	return false
}
func (b BattleReport) Losses(owner int) (own, enemy int) {
	for _, a := range b.Armies {
		if a.Owner == owner {
			own += a.Lost
		} else {
			enemy += a.Lost
		}
	}
	return
}
func (s *State) currentMilitaryReport(id int) *MilitaryReport {
	for i := len(s.Reports) - 1; i >= 0 && s.Reports[i].Month == s.Calendar.Elapsed; i-- {
		if s.Reports[i].Owner == id {
			return &s.Reports[i]
		}
	}
	s.Reports = append(s.Reports, MilitaryReport{Owner: id, Month: s.Calendar.Elapsed})
	return &s.Reports[len(s.Reports)-1]
}
func (s *State) recordArmyChange(id, slot int, before Army, after int, cause ArmyChangeCause) {
	if before.People == after {
		return
	}
	r := s.currentMilitaryReport(id)
	r.Changes = append(r.Changes, ArmyChange{slot, before.Position, before.People, after, cause})
}

// MilitaryHistory returns private, independently owned snapshots, including
// current-month reorganizations. Only completed months are autosaved by play.
func (s *State) MilitaryHistory(id int) []MilitaryReport {
	if id < 0 || id >= TribeCount {
		return nil
	}
	var history []MilitaryReport
	for _, r := range s.Reports {
		if r.Owner != id {
			continue
		}
		r.Changes = append([]ArmyChange(nil), r.Changes...)
		r.Battles = append([]BattleReport(nil), r.Battles...)
		for i := range r.Battles {
			r.Battles[i].Armies = append([]BattleArmy(nil), r.Battles[i].Armies...)
		}
		history = append(history, r)
	}
	return history
}
