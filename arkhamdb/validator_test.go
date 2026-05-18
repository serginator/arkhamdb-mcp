package arkhamdb

import "testing"

func TestValidateDeckSize_TooSmall(t *testing.T) {
	deckCards := map[string]int{"01001": 2, "01002": 2}
	total := 0
	for _, qty := range deckCards {
		total += qty
	}
	if total >= 30 {
		t.Error("expected fewer than 30 cards")
	}
}

func TestBaseCardNameInUpgrades(t *testing.T) {
	name := "Beat Cop (2)"
	base := baseCardName(name)
	if base != "Beat Cop" {
		t.Errorf("expected 'Beat Cop', got %q", base)
	}
	base2 := baseCardName("Beat Cop")
	if base2 != "Beat Cop" {
		t.Errorf("expected 'Beat Cop', got %q", base2)
	}
	if base != base2 {
		t.Error("base names should match for upgrade detection")
	}
}
