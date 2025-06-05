package models

import "fmt"

type MarketItem struct {
	ID       string
	IDObject string
	Level    string
	Gold     float64
	Value    float64
	Rarity   string
}

// Diff returns the difference between value and gold.
func (lgr *MarketItem) Diff() float64 {
	return lgr.Value - lgr.Gold
}

func (lgr *MarketItem) String() string {
	return fmt.Sprintf("Level %s ==> %.0f 🪙 | value: %.0f | diff: %.0f | id: %s | Rarety: %s", lgr.Level, lgr.Gold, lgr.Value, lgr.Diff(), lgr.ID, lgr.Rarity)
}
