package models

// ItemStatsResponse models the JSON payload from /api/item/stats-v2/{id}.
type ItemStatsResponse struct {
	Success bool      `json:"success"`
	Item    ItemStats `json:"item"`
}

// ItemStats contains item details returned by stats-v2.
type ItemStats struct {
	ID    int     `json:"id"`
	Value float64 `json:"value"`
}
