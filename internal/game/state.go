package game

import (
	"fmt"
	"math/rand"
)

type Point struct{ X, Y int }

func (p Point) Valid() bool { return p.X >= 0 && p.X < Width && p.Y >= 0 && p.Y < Height }
func (p Point) Neighbors() [4]Point {
	return [4]Point{{p.X, p.Y - 1}, {p.X - 1, p.Y}, {p.X + 1, p.Y}, {p.X, p.Y + 1}}
}

type Terrain uint8

const (
	Grassland Terrain = iota
	Forest
	Mountain
	River
	Ocean
)

func (t Terrain) Passable() bool { return t == Grassland || t == Forest }
func (t Terrain) String() string {
	return [...]string{"草原", "森林", "山地", "河流", "海洋"}[t]
}
func (t Terrain) Symbol() byte { return [...]byte{'.', 'f', '^', '=', '~'}[t] }

// Zero means no farm. Other values encode 0..FarmMonths completed advances.
type FarmStage uint8

const (
	FarmNone    FarmStage = 0
	FarmPlanned FarmStage = 1
	FarmReady   FarmStage = FarmStage(FarmMonths + 1)
)

func (f FarmStage) Progress() int {
	if f == FarmNone {
		return 0
	}
	return int(f) - 1
}

func (f FarmStage) advance() FarmStage {
	if f == FarmNone || f == FarmReady {
		return f
	}
	return f + 1
}

type Tile struct {
	Terrain    Terrain
	Owner      int
	Farm       FarmStage
	Settlement bool // Physical founding settlement; it does not move with the capital.
	RestUntil  int  // Captured facilities resume when the calendar reaches this month.
}

type Stock struct{ Food, Wood int }

type Tribe struct {
	ID             int
	Name           string
	Capital        Point
	Population     int
	Soldiers       int
	Food           int
	Wood           int
	AP             int
	GrowthProgress int // Fractional person in GrowthScale units; not a month counter.
	HungerMonths   int
	ForestPriority int // Minimum forests staffed before the usual farm-first allocation.
	Armies         [MaxArmies]Army
	Defeated       bool // Military defeat retains the surviving population; it is not a death toll.
	Explored       [Height][Width]bool
	LastChange     Stock
	monthStart     Stock
}

func (t Tribe) Alive() bool { return t.Population > 0 && !t.Defeated }

func (t Tribe) Status() string {
	if t.Defeated {
		return "战败"
	}
	if t.Population == 0 {
		return "灭亡"
	}
	return "存活"
}

func newTribe(id int, capital Point) Tribe {
	t := Tribe{
		ID: id, Name: fmt.Sprintf("部落%d", id+1), Capital: capital,
		Population: InitialPopulation, Soldiers: InitialSoldiers,
		Food: InitialFood, Wood: InitialWood, AP: MonthlyAP,
		monthStart: Stock{InitialFood, InitialWood},
	}
	t.Explored[capital.Y][capital.X] = true
	return t
}

// Event visibility is captured at emission, not recomputed after later exploration.
type Event struct {
	Month    int
	Audience uint8
	Text     string
}

// State has one owner and one local RNG. It is not shared between goroutines.
type State struct {
	Seed         int64
	Calendar     Calendar
	Tiles        [Height][Width]Tile
	Tribes       [TribeCount]Tribe
	Events       []Event
	Reports      []MilitaryReport
	ReportsSince int // First month with structured records; old saves start at load time.
	rng          *rand.Rand
}

func (s *State) AllExtinct() bool {
	for _, t := range s.Tribes {
		if t.Alive() {
			return false
		}
	}
	return true
}

func (s *State) Validate() error {
	if s.Calendar.Elapsed < 0 || s.rng == nil {
		return fmt.Errorf("无效日历或随机数状态")
	}
	for id, t := range s.Tribes {
		if err := s.validateTribe(id, t); err != nil {
			return err
		}
	}
	for y, row := range s.Tiles {
		for x, tile := range row {
			if err := s.validateTile(Point{x, y}, tile); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *State) validateTribe(id int, t Tribe) error {
	if err := validatePopulationRecords(t); err != nil {
		return err
	}
	if err := validateWorkPlan(t); err != nil {
		return err
	}
	if err := s.validateArmies(id); err != nil {
		return err
	}
	if t.ID != id || !t.Capital.Valid() {
		return fmt.Errorf("部落身份或首都无效: %d", id)
	}
	if t.Population < 0 || t.Soldiers < 0 || t.Soldiers > t.Population {
		return fmt.Errorf("部落%d人口或士兵无效", id+1)
	}
	if t.Food < 0 || t.Wood < 0 || t.AP < 0 || t.AP > MonthlyAP {
		return fmt.Errorf("部落%d资源或行动点无效", id+1)
	}
	if !t.Alive() {
		if t.Food != 0 || t.Wood != 0 || t.AP != 0 || t.Soldiers != 0 {
			return fmt.Errorf("灭亡部落未清理")
		}
		return nil
	}
	if s.Tiles[t.Capital.Y][t.Capital.X].Owner != id {
		return fmt.Errorf("存活部落缺少首都")
	}
	return nil
}

func (s *State) validateTile(p Point, tile Tile) error {
	if tile.Terrain > Ocean || tile.Farm > FarmReady {
		return fmt.Errorf("地形或农田阶段无效: %v", p)
	}
	if tile.Owner < Unowned || tile.Owner >= TribeCount {
		return fmt.Errorf("归属无效: %v", p)
	}
	if tile.RestUntil < 0 || tile.RestUntil > s.Calendar.Elapsed+1 {
		return fmt.Errorf("占领休整记录无效: %v", p)
	}
	if tile.Farm != FarmNone && (tile.Owner == Unowned || tile.Terrain != Grassland) {
		return fmt.Errorf("农田地块无效: %v", p)
	}
	if tile.Owner == Unowned {
		return nil
	}
	t := s.Tribes[tile.Owner]
	if !tile.Terrain.Passable() || !t.Alive() || !t.Explored[p.Y][p.X] {
		return fmt.Errorf("领土或探索记录无效: %v", p)
	}
	return nil
}
