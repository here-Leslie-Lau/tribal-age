package game

import "fmt"

type ArmyOrderKind uint8

const (
	Hold ArmyOrderKind = iota
	March
	Attack
	Occupy
)

func (k ArmyOrderKind) String() string {
	switch k {
	case Hold:
		return "驻守"
	case March:
		return "行军"
	case Attack:
		return "进攻"
	case Occupy:
		return "占领"
	}
	return "未知军令"
}

type ArmyOrder struct {
	Kind   ArmyOrderKind
	Target Point
}
type Army struct {
	People     int
	Position   Point
	Order      ArmyOrder
	Unsupplied int
}
type ArmyInfo struct {
	Owner, Slot, People int
	Position            Point
	Supply              SupplyLevel
}

func (t Tribe) ReserveSoldiers() int {
	n := t.Soldiers
	for _, a := range t.Armies {
		n -= a.People
	}
	return n
}

func (s *State) militaryActor(id, slot int) error {
	if id < 0 || id >= TribeCount || slot < 0 || slot >= MaxArmies {
		return fmt.Errorf("部落或军队编号无效")
	}
	if !s.Tribes[id].Alive() {
		return fmt.Errorf("部落已经退出竞争")
	}
	return nil
}
func (s *State) SetArmySize(id, slot, people int) error {
	if err := s.militaryActor(id, slot); err != nil {
		return err
	}
	t := &s.Tribes[id]
	a := t.Armies[slot]
	if people < 0 || people > MaxArmyPeople || people > t.ReserveSoldiers()+a.People {
		return fmt.Errorf("人数超出可用士兵；需要先训练，不能凭空补兵")
	}
	if a.People == people {
		return nil
	}
	if err := s.checkArmyResize(id, a); err != nil {
		return err
	}
	t.AP -= ActionAP
	if people == 0 {
		t.Armies[slot] = Army{}
	} else {
		a.People, a.Position = people, t.Capital
		t.Armies[slot] = a
	}
	s.revealArmies(id)
	s.emitOwn(id, fmt.Sprintf("军队%c编组为%d人，未编组士兵%d。", 'A'+slot, people, t.ReserveSoldiers()))
	return nil
}
func (s *State) SetArmyOrder(id, slot int, order ArmyOrder) error {
	if err := s.militaryActor(id, slot); err != nil {
		return err
	}
	a := &s.Tribes[id].Armies[slot]
	if a.People == 0 {
		return fmt.Errorf("该军队尚未编组")
	}
	if err := s.checkArmyOrder(id, *a, order); err != nil {
		return err
	}
	if order.Kind == Hold {
		order = ArmyOrder{}
	} else if order.Kind == Occupy {
		order.Target = a.Position
	}
	if a.Order == order {
		return nil
	}
	a.Order = order
	s.emitOwn(id, fmt.Sprintf("军队%c军令：%s；月底执行，可以修改。", 'A'+slot, order.Kind))
	return nil
}
func (s *State) checkArmyOrder(id int, a Army, o ArmyOrder) error {
	switch o.Kind {
	case Hold:
		return nil
	case Occupy:
		tile := s.Tiles[a.Position.Y][a.Position.X]
		if tile.Owner == id || tile.Owner == Unowned {
			return fmt.Errorf("占领军令只能接收军队脚下的敌方领土")
		}
		return nil
	case March, Attack:
		return s.checkMovingOrder(id, a, o)
	default:
		return fmt.Errorf("未知军令")
	}
}
func manhattan(a, b Point) int { return absDistance(a.X-b.X) + absDistance(a.Y-b.Y) }
func absDistance(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
func (s *State) hostileAt(id int, p Point) bool {
	for owner, t := range s.Tribes {
		if owner == id {
			continue
		}
		for _, a := range t.Armies {
			if a.People > 0 && a.Position == p {
				return true
			}
		}
	}
	return false
}
func (s *State) canSeeArmy(id int, p Point) bool {
	if !s.Tribes[id].Alive() {
		return false
	}
	for _, a := range s.Tribes[id].Armies {
		if a.People > 0 && manhattan(a.Position, p) <= 1 {
			return true
		}
	}
	return s.nearOwned(id, p)
}
func (s *State) nearOwned(id int, p Point) bool {
	if s.Tiles[p.Y][p.X].Owner == id {
		return true
	}
	for _, n := range p.Neighbors() {
		if n.Valid() && s.Tiles[n.Y][n.X].Owner == id {
			return true
		}
	}
	return false
}
func (s *State) visibleArmies(id int) []ArmyInfo {
	var out []ArmyInfo
	for owner, t := range s.Tribes {
		for slot, a := range t.Armies {
			if a.People > 0 && (owner == id || s.canSeeArmy(id, a.Position)) {
				out = append(out, ArmyInfo{Owner: owner, Slot: slot, People: a.People, Position: a.Position, Supply: s.supplyAt(owner, a.Position)})
			}
		}
	}
	return out
}
func (s *State) revealArmies(id int) {
	for _, a := range s.Tribes[id].Armies {
		if a.People == 0 {
			continue
		}
		s.Tribes[id].Explored[a.Position.Y][a.Position.X] = true
		for _, n := range a.Position.Neighbors() {
			if n.Valid() {
				s.Tribes[id].Explored[n.Y][n.X] = true
			}
		}
	}
}
func (s *State) validateArmies(id int) error {
	for _, a := range s.Tribes[id].Armies {
		if err := s.validateArmyRecord(id, a); err != nil {
			return err
		}
	}
	if s.Tribes[id].ReserveSoldiers() < 0 {
		return fmt.Errorf("军队人数超过部落士兵")
	}
	return nil
}
func (s *State) validateArmyRecord(id int, a Army) error {
	if a.People == 0 {
		if a != (Army{}) {
			return fmt.Errorf("空军队记录未清理")
		}
		return nil
	}
	if err := validateArmyBasics(s.Tribes[id], a); err != nil {
		return err
	}
	if !s.Tiles[a.Position.Y][a.Position.X].Terrain.Passable() || !s.Tribes[id].Explored[a.Position.Y][a.Position.X] {
		return fmt.Errorf("军队所在位置无效")
	}
	if s.hostileAt(id, a.Position) {
		return fmt.Errorf("敌对军队不能共处一格")
	}
	return s.validateOrderRecord(id, a)
}
func (s *State) validateOrderRecord(id int, a Army) error {
	o := a.Order
	switch o.Kind {
	case Hold:
		if o.Target != (Point{}) {
			return fmt.Errorf("驻守军令记录无效")
		}
	case Occupy:
		if o.Target != a.Position {
			return fmt.Errorf("占领军令记录无效")
		}
	case March, Attack:
		return s.checkArmyRoute(id, a.Position, o.Target)
	default:
		return fmt.Errorf("军令记录无效")
	}
	return nil
}

func (s *State) checkArmyResize(id int, a Army) error {
	t := s.Tribes[id]
	if t.AP < ActionAP {
		return fmt.Errorf("编组需要 %d 行动点", ActionAP)
	}
	if a.People > 0 && a.Position != t.Capital {
		return fmt.Errorf("军队必须返回首都才能调整兵数")
	}
	if s.hostileAt(id, t.Capital) {
		return fmt.Errorf("首都有敌军，不能编组")
	}
	return nil
}
func validateArmyBasics(t Tribe, a Army) error {
	if !t.Alive() || a.People < 0 || a.People > MaxArmyPeople || a.Unsupplied < 0 {
		return fmt.Errorf("军队记录无效")
	}
	if !a.Position.Valid() {
		return fmt.Errorf("军队位置越界")
	}
	return nil
}
func (s *State) checkArmyRoute(id int, from, to Point) error {
	if !to.Valid() || manhattan(from, to) != 1 {
		return fmt.Errorf("军令目标必须是上下左右相邻一格")
	}
	if !s.Tribes[id].Explored[to.Y][to.X] {
		return fmt.Errorf("军令目标尚未探索")
	}
	if !s.Tiles[to.Y][to.X].Terrain.Passable() {
		return fmt.Errorf("该地形不能通行")
	}
	return nil
}
func (s *State) checkMovingOrder(id int, a Army, o ArmyOrder) error {
	if err := s.checkArmyRoute(id, a.Position, o.Target); err != nil {
		return err
	}
	enemy := s.hostileAt(id, o.Target)
	if o.Kind == Attack && !enemy {
		return fmt.Errorf("目标没有可见敌军")
	}
	if o.Kind == March && enemy {
		return fmt.Errorf("目标有敌军，请使用进攻军令")
	}
	return nil
}
