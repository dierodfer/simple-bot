package models

type MarketItem struct {
	ID       string
	IDObject string
	Level    string
	Gold     float64
	Value    float64
}

// Diff returns the difference between value and gold.
func (lgr *MarketItem) Diff() float64 {
	return lgr.Value - lgr.Gold
}
