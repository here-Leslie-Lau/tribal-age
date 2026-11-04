package game

import "testing"

func TestAIPreparesFoodForGrowthDuringFarmLeadTime(t *testing.T) {
	s := growthScenario()
	s.Tribes[0].Population, s.Tribes[0].Food, s.Tribes[0].Wood = 38, 100, 20
	ownTile(s, 0, Point{2, 2}, Forest, FarmNone)
	ownTile(s, 0, Point{3, 2}, Forest, FarmNone)
	ownTile(s, 0, Point{4, 3}, Grassland, FarmNone)
	v, _ := s.View(0)
	a, ok := ChooseAction(v)
	if !ok || a != (Action{BuildFarm, Point{4, 3}}) {
		t.Fatalf("AI waited for population 41 instead of preparing food: %+v", a)
	}
}

func TestAIExpandsBeforeSpaceSuppressesGrowth(t *testing.T) {
	s := growthScenario()
	s.Tribes[0].Population, s.Tribes[0].Food, s.Tribes[0].Wood = 34, 100, 20
	ownTile(s, 0, Point{2, 2}, Forest, FarmNone)
	s.Tribes[0].Explored[2][3] = true
	v, _ := s.View(0)
	a, ok := ChooseAction(v)
	if !ok || a != (Action{Claim, Point{3, 2}}) {
		t.Fatalf("AI waited until full capacity: %+v", a)
	}
}
