package game

import "fmt"

type ActionKind uint8

const (
	Explore ActionKind = iota
	Claim
	BuildFarm
	Train
)

type Action struct {
	Kind   ActionKind
	Target Point
}

// Apply validates everything before the first mutation; it never uses the RNG.
func (s *State) Apply(id int, action Action) error {
	if err := s.checkAction(id, action); err != nil {
		return err
	}
	t := &s.Tribes[id]
	t.AP -= ActionAP
	p := action.Target
	switch action.Kind {
	case Explore:
		t.Explored[p.Y][p.X] = true
		s.emitAt(id, p, fmt.Sprintf("%s探索 (%d,%d)：%s。", t.Name, p.X, p.Y, s.Tiles[p.Y][p.X].Terrain))
	case Claim:
		t.Wood -= ClaimWood
		s.Tiles[p.Y][p.X].Owner = id
		s.emitAt(id, p, fmt.Sprintf("%s占领 (%d,%d)。", t.Name, p.X, p.Y))
	case BuildFarm:
		t.Wood -= FarmWood
		s.Tiles[p.Y][p.X].Farm = FarmPlanned
		s.emitAt(id, p, fmt.Sprintf("%s在 (%d,%d) 开工农田，进度 0/%d。", t.Name, p.X, p.Y, FarmMonths))
	case Train:
		t.Food -= TrainingFood
		t.Soldiers += TrainingSize
		s.emitOwn(id, fmt.Sprintf("%s训练了 %d 名士兵。", t.Name, TrainingSize))
	}
	return nil
}

func (s *State) checkAction(id int, a Action) error {
	if id < 0 || id >= TribeCount {
		return fmt.Errorf("部落编号无效")
	}
	t := s.Tribes[id]
	if !t.Alive() {
		return fmt.Errorf("部落已经%s", t.Status())
	}
	if t.AP < ActionAP {
		return fmt.Errorf("行动点不足，需要 %d 点", ActionAP)
	}
	if a.Kind > Train {
		return fmt.Errorf("未知行动")
	}
	if a.Kind == Train {
		if s.hostileAt(id, t.Capital) {
			return fmt.Errorf("首都有敌军，不能训练")
		}
		return checkTraining(t)
	}
	if !a.Target.Valid() {
		return fmt.Errorf("坐标越界：x 为 0–%d，y 为 0–%d", Width-1, Height-1)
	}
	switch a.Kind {
	case Explore:
		if t.Explored[a.Target.Y][a.Target.X] {
			return fmt.Errorf("目标已探索")
		}
		if !s.adjacentTo(id, a.Target) {
			return fmt.Errorf("探索目标必须邻接自有领土")
		}
	case Claim:
		return s.checkClaim(id, a.Target)
	case BuildFarm:
		return s.checkFarm(id, a.Target)
	}
	return nil
}

func checkTraining(t Tribe) error {
	if t.Population-t.Soldiers < TrainingSize {
		return fmt.Errorf("平民不足，需要 %d 人", TrainingSize)
	}
	if t.Food < TrainingFood {
		return fmt.Errorf("粮食不足，需要 %d", TrainingFood)
	}
	return nil
}

func (s *State) checkClaim(id int, p Point) error {
	t := s.Tribes[id]
	if !t.Explored[p.Y][p.X] {
		return fmt.Errorf("目标尚未探索")
	}
	tile := s.Tiles[p.Y][p.X]
	if tile.Owner != Unowned {
		return fmt.Errorf("目标已有归属")
	}
	if !tile.Terrain.Passable() {
		return fmt.Errorf("目标地形不可通行，不能占领")
	}
	if !s.adjacentTo(id, p) {
		return fmt.Errorf("占领目标必须邻接自有领土")
	}
	if s.hostileAt(id, p) {
		return fmt.Errorf("目标有敌军，不能由平民占领")
	}
	if t.Wood < ClaimWood {
		return fmt.Errorf("木材不足，占领需要 %d", ClaimWood)
	}
	return nil
}

func (s *State) checkFarm(id int, p Point) error {
	tile := s.Tiles[p.Y][p.X]
	if tile.Owner != id {
		return fmt.Errorf("只能在自有领土建农田")
	}
	if s.hostileAt(id, p) {
		return fmt.Errorf("目标有敌军，不能建造")
	}
	if tile.Terrain != Grassland {
		return fmt.Errorf("农田只能建在草原")
	}
	if tile.Farm != FarmNone {
		return fmt.Errorf("该地已有在建或完工农田")
	}
	if s.Tribes[id].Wood < FarmWood {
		return fmt.Errorf("木材不足，农田需要 %d", FarmWood)
	}
	return nil
}

func (s *State) adjacentTo(id int, p Point) bool {
	for _, n := range p.Neighbors() {
		if n.Valid() && s.Tiles[n.Y][n.X].Owner == id {
			return true
		}
	}
	return false
}
