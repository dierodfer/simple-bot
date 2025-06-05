package models

import "fmt"

type MarketItem struct {
	ID       string
	IDObject string
	Level    string
	Rarity   string
	Type     string
	Gold     float64
	Value    float64
}

// Diff returns the difference between value and gold.
func (lgr *MarketItem) Diff() float64 {
	return lgr.Value - lgr.Gold
}

func (item *MarketItem) String() string {
	return fmt.Sprintf("Level %s ==> %.0f 🪙 | value: %.0f | diff: %.0f | id: %s | %s -- %s", item.Level, item.Gold, item.Value, item.Diff(), item.ID, item.Rarity, item.Type)
}
