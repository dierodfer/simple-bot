package models

import (
	"fmt"
	"strings"
)

const goodProfitRatio = 0.25

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

// ProfitToValueRatio returns profit/value ratio in range [-inf, +inf].
// Returns 0 when Value is 0 to avoid division by zero.
func (item *MarketItem) ProfitToValueRatio() float64 {
	if item.Value == 0 {
		return 0
	}
	return item.Diff() / item.Value
}

// IsWeapon reports whether the item type is weapon.
func (item *MarketItem) IsWeapon() bool {
	return strings.EqualFold(item.Type, "Weapon")
}

// IsCelestial reports whether the item rarity is celestial.
func (item *MarketItem) IsCelestial() bool {
	return strings.EqualFold(item.Rarity, "Celestial")
}

func (item *MarketItem) hasGoodProfitRatio() bool {
	return item.Diff() > 0 && item.ProfitToValueRatio() >= goodProfitRatio
}

// IsGoodWeaponDeal reports if a weapon has a good value-profit relationship.
// Current rule: weapon with positive profit and ratio >= 25%.
func (item *MarketItem) IsGoodWeaponDeal() bool {
	return item.IsWeapon() && item.hasGoodProfitRatio()
}

// IsGoodCelestialDeal reports if a celestial item has a good value-profit relationship.
func (item *MarketItem) IsGoodCelestialDeal() bool {
	return item.IsCelestial() && item.hasGoodProfitRatio()
}

func (item *MarketItem) String() string {
	return fmt.Sprintf("Level %s ==> %.0f 🪙 | value: %.0f | profit: %.0f | id: %s | %s -- %s", item.Level, item.Gold, item.Value, item.Diff(), item.ID, item.Rarity, item.Type)
}
