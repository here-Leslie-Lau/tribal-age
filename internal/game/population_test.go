package game

import (
	"reflect"
	"strings"
	"testing"
)

func growthScenario() *State {
	s := scenarioState()
	ownTile(s, 0, Point{2, 3}, Grassland, FarmReady)
	ownTile(s, 0, Point{3, 3}, Grassland, FarmReady)
	s.Tribes[0].Food = 23
	s.Tribes[0].monthStart = Stock{23, 20}
	return s
}

func TestPopulationGrowthHandExample(t *testing.T) {
	s := growthScenario()
	for i, want := range []struct{ food, population, progress int }{{24, 30, 375}, {25, 30, 750}, {26, 31, 125}, {30, 31, 425}} {
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		tribe := s.Tribes[0]
		if tribe.Food != want.food || tribe.Population != want.population || tribe.GrowthProgress != want.progress || tribe.Soldiers != 5 || tribe.HungerMonths != 0 || tribe.AP != 2 {
			t.Fatalf("month %d: wrong population settlement: %+v", i, tribe)
		}
		if err := s.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	if !strings.Contains(strings.Join(s.VisibleEvents(0, 0), "\n"), "部落1人口自然增长") {
		t.Fatal("growth not explained")
	}
	if strings.Contains(strings.Join(s.VisibleEvents(1, 0), "\n"), "部落1人口自然增长") {
		t.Fatal("private growth event leaked")
	}
	if s.rng.Int63() != scenarioState().rng.Int63() {
		t.Fatal("population rules used randomness")
	}
}

func TestCarryingCapacityAndGrowthThresholds(t *testing.T) {
	s := scenarioState()
	for count, want := range []int{30, 35, 40} {
		if count > 0 {
			ownTile(s, 0, Point{2 + count, 3}, Forest, FarmNone)
		}
		v, err := s.View(0)
		if err != nil {
			t.Fatal(err)
		}
		if CarryingCapacity(v) != want {
			t.Fatalf("%d extra tiles: wrong capacity", count)
		}
	}
	for _, tt := range []struct {
		name                     string
		food, population, hunger int
		blocked                  string
	}{
		{"below reserve", 23, 30, 0, "存粮不足"}, {"exact reserve", 24, 30, 0, ""},
		{"at capacity", 100, 35, 0, "承载力"}, {"over capacity", 100, 36, 0, "承载力"},
		{"grace month", 100, 30, 1, "缺粮"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := growthScenario()
			s.Tribes[0].Food = tt.food
			s.Tribes[0].Population = tt.population
			s.Tribes[0].HungerMonths = tt.hunger
			v, err := s.View(0)
			if err != nil {
				t.Fatal(err)
			}
			g := GrowthFor(v)
			if g.Capacity != 35 || (tt.population == 30 && g.Reserve != 24) {
				t.Fatalf("wrong growth limits: %+v", g)
			}
			if tt.blocked == "" {
				if g.Reason != "" {
					t.Fatalf("eligible growth blocked: %s", g.Reason)
				}
			} else if !strings.Contains(g.Reason, tt.blocked) {
				t.Fatalf("wrong growth reason: %s", g.Reason)
			}
		})
	}
}

func TestLowReservePausesGrowthAndLandClearsProgress(t *testing.T) {
	t.Run("training spends reserve", func(t *testing.T) {
		s := growthScenario()
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		if err := s.Apply(0, Action{Kind: Train}); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].GrowthProgress != 375 {
			t.Fatal("an action advanced or reset the monthly counter")
		}
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].GrowthProgress != 375 || s.Tribes[0].Population != 30 || s.Tribes[0].HungerMonths != 0 {
			t.Fatal("low reserves did not stop growth independently of starvation")
		}
	})
	for _, population := range []int{35, 36} {
		s := growthScenario()
		s.Tribes[0].Population = population
		s.Tribes[0].Food = 100
		s.Tribes[0].GrowthProgress = 2
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].Population != population || s.Tribes[0].GrowthProgress != 0 {
			t.Fatal("capacity must stop growth, not kill existing population or bank progress")
		}
		if err := s.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHungerGraceRecoveryAndContinuousLoss(t *testing.T) {
	s := scenarioState()
	s.Tribes[0].Food = 3
	s.Tribes[0].GrowthProgress = 2
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].Population != 30 || s.Tribes[0].Food != 0 || s.Tribes[0].HungerMonths != 1 || s.Tribes[0].GrowthProgress != 0 {
		t.Fatal("first shortage did not grant grace and interrupt growth")
	}
	// A hand-built recovery month with exactly enough food, not enough growth reserve.
	s.Tribes[0].Food = 7
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].HungerMonths != 0 || s.Tribes[0].GrowthProgress != 0 {
		t.Fatal("normal feeding did not clear hunger independently of growth")
	}
	for _, want := range []struct{ population, hunger int }{{30, 1}, {23, 2}, {17, 3}} {
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].Population != want.population || s.Tribes[0].HungerMonths != want.hunger || s.Tribes[0].Soldiers != 5 {
			t.Fatalf("wrong repeated shortage: %+v", s.Tribes[0])
		}
		if err := s.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWinterGrowthAndNextMonthWorkers(t *testing.T) {
	t.Run("winter stores permit growth", func(t *testing.T) {
		s := growthScenario()
		s.Calendar.Elapsed = 9
		s.Tribes[0].Food = 60
		for i := 0; i < 3; i++ {
			if err := s.EndMonth(Interactive); err != nil {
				t.Fatal(err)
			}
		}
		if s.Tribes[0].Population != 31 || s.Tribes[0].Food != 39 || s.Calendar.Month() != 3 {
			t.Fatal("winter wrongly required positive net food")
		}
	})
	t.Run("extra workers start next month", func(t *testing.T) {
		s := scenarioState()
		s.Calendar.Elapsed = 2
		s.Tribes[0].Population = 34
		s.Tribes[0].Food = 100
		s.Tribes[0].GrowthProgress = 900
		for _, p := range []Point{{2, 3}, {3, 3}, {4, 3}, {2, 2}, {3, 2}, {4, 2}} {
			ownTile(s, 0, p, Grassland, FarmReady)
		}
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].Population != 35 || s.Tribes[0].WorkSlots() != 6 || s.Tribes[0].Food != 112 {
			t.Fatal("new workers or new food demand applied retroactively")
		}
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].Food != 140 {
			t.Fatal("new workers not available next month")
		}
	})
}

func TestPopulationRecordsValidateAndClearOnExtinction(t *testing.T) {
	for _, change := range []func(*State){
		func(s *State) { s.Tribes[0].GrowthProgress = -1 }, func(s *State) { s.Tribes[0].GrowthProgress = GrowthScale },
		func(s *State) { s.Tribes[0].HungerMonths = -1 }, func(s *State) { s.Tribes[0].GrowthProgress = 1; s.Tribes[0].HungerMonths = 1 },
	} {
		s := scenarioState()
		change(s)
		if s.Validate() == nil {
			t.Fatal("invalid population records accepted")
		}
	}
	s := scenarioState()
	s.Tribes[0].Population = 1
	s.Tribes[0].Soldiers = 1
	s.Tribes[0].Food = 0
	s.Tribes[0].Wood = 0
	s.Tribes[0].GrowthProgress = 2
	for i := 0; i < 2; i++ {
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
	}
	if s.Tribes[0].Alive() || s.Tribes[0].GrowthProgress != 0 || s.Tribes[0].HungerMonths != 0 {
		t.Fatal("extinction records not cleared")
	}
	v, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	if CarryingCapacity(v) != 0 || GrowthFor(v).Reason == "" {
		t.Fatal("extinct tribe can grow")
	}
	for _, change := range []func(*State){func(s *State) { s.Tribes[0].GrowthProgress = 1 }, func(s *State) { s.Tribes[0].HungerMonths = 1 }} {
		dead := *s
		change(&dead)
		if dead.Validate() == nil {
			t.Fatal("extinct population records accepted")
		}
	}
	before := s.Tribes[0]
	if err := s.EndMonth(Simulation); err != nil {
		t.Fatal(err)
	}
	// Inventory delta becomes zero next month, but extinction and population records remain final.
	before.LastChange = s.Tribes[0].LastChange
	if !reflect.DeepEqual(before, s.Tribes[0]) {
		t.Fatal("extinct tribe was revived or changed")
	}
}

func TestRejectedActionDoesNotChangeGrowthRecords(t *testing.T) {
	s := growthScenario()
	s.Tribes[0].GrowthProgress = 2
	before := *s
	if err := s.Apply(0, Action{BuildFarm, Point{2, 3}}); err == nil {
		t.Fatal("repeat farm accepted")
	}
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("failed action changed population progress")
	}
}

func TestStarvationChangesOnlyFollowingMonthLabor(t *testing.T) {
	s := productionScenario()
	s.Calendar.Elapsed = 9
	s.Tribes[0].Food = 0
	for _, want := range []struct{ population, wood int }{{30, 28}, {23, 36}, {17, 42}} {
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].Population != want.population || s.Tribes[0].Wood != want.wood {
			t.Fatalf("starvation changed production retroactively: %+v", s.Tribes[0])
		}
		if err := s.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReferenceSimulationPopulationDevelops(t *testing.T) {
	s := New(42)
	for month := 0; month < 120; month++ {
		if err := s.EndMonth(Simulation); err != nil {
			t.Fatal(err)
		}
		if err := s.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for id, tribe := range s.Tribes {
		v, err := s.View(id)
		if err != nil {
			t.Fatal(err)
		}
		if !tribe.Alive() || tribe.Population <= 30 || tribe.Population > CarryingCapacity(v) || tribe.HungerMonths != 0 {
			t.Fatalf("reference tribe did not develop sustainably: %+v", tribe)
		}
	}
}
