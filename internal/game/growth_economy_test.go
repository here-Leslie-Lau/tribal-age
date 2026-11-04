package game

import "testing"

func TestEngineMatchesIndependentFixedCapacityGrowthCurves(t *testing.T) {
	for _, tt := range []struct{ population, capacity, fullAt int }{{30, 35, 30}, {30, 40, 40}, {100, 120, 47}, {200, 240, 56}} {
		s := sizedGrowthState(t, tt.population, tt.capacity, 100000, 0)
		fullAt := 0
		for month := 1; month <= 120; month++ {
			if err := s.EndMonth(Interactive); err != nil {
				t.Fatal(err)
			}
			if s.Tribes[0].Population == tt.capacity && fullAt == 0 {
				fullAt = month
			}
			if err := s.Validate(); err != nil {
				t.Fatal(err)
			}
		}
		if fullAt != tt.fullAt || s.Tribes[0].Population != tt.capacity || s.Tribes[0].GrowthProgress != 0 {
			t.Fatalf("P/K=%d/%d full at %d, want %d", tt.population, tt.capacity, fullAt, tt.fullAt)
		}
	}
}

func TestReferenceTribesReachLargePopulationsWithoutHunger(t *testing.T) {
	s := New(42)
	for month := 0; month < 240; month++ {
		if err := s.EndMonth(Simulation); err != nil {
			t.Fatal(err)
		}
		if err := s.Validate(); err != nil {
			t.Fatal(err)
		}
		for _, tribe := range s.Tribes {
			if !tribe.Alive() || tribe.HungerMonths != 0 {
				t.Fatalf("month %d: growth outpaced food planning: tribe=%d population=%d hunger=%d", month+1, tribe.ID, tribe.Population, tribe.HungerMonths)
			}
		}
	}
	for id, tribe := range s.Tribes {
		v, _ := s.View(id)
		if tribe.Population < 200 || tribe.Population > CarryingCapacity(v) {
			t.Fatalf("reference growth failed to reach sustainable large population: %d", tribe.Population)
		}
	}
}
