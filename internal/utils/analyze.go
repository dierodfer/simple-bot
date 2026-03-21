package utils

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	keystore "simple-bot/internal/database"
	"simple-bot/internal/models"
)

// Profit thresholds for market analysis decisions.
const (
	ProfitThresholdBuy       = 15000
	ProfitThresholdHighlight = 10000
	ProfitThresholdShow      = 1000
	CelestialMaxLoss         = -500000
)

// MarketOptions holds the parameters for a market analysis run.
type MarketOptions struct {
	URLListItems models.ListItemsURL
	Threads      int
	MinLevel     int
	MaxLevel     int
	LevelRange   int
	MaxPages     int
	RecentItems  bool
	ShowAll      bool
}

// AnalyzeInspectParallel inspects item values in parallel across the given ID range
// and stores results in the provided key-value store.
func AnalyzeInspectParallel(httpClient *HTTPClient, store keystore.KeyValueStore, baseURL string, threads, startID, endID int) {
	var wg sync.WaitGroup
	idCh := make(chan int, threads)

	for w := 0; w < threads; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range idCh {
				id := strconv.Itoa(i)
				value, err := inspectItemValue(httpClient, baseURL, id)
				if err != nil {
					log.Printf("Error inspecting item %s: %v", id, err)
					continue
				}
				if value > 0 {
					log.Printf("Item %s value: %.2f\n", id, value)
					if err := store.Set(id, fmt.Sprintf("%.0f", value)); err != nil {
						log.Printf("Error saving item %s: %v", id, err)
					}
				}
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}

	for i := startID; i <= endID; i++ {
		idCh <- i
	}
	close(idCh)
	wg.Wait()
}

// AnalyzeMarket scans the market across level ranges and identifies profitable items.
func AnalyzeMarket(httpClient *HTTPClient, store keystore.KeyValueStore, baseURL string, opts MarketOptions) {
	var wg sync.WaitGroup
	levelCh := make(chan int, opts.Threads)

	for i := 0; i < opts.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for level := range levelCh {
				time.Sleep(time.Duration(1+rand.Intn(10)) * time.Second)
				processLevelRange(httpClient, store, baseURL, opts, level)
			}
		}()
	}

	for level := opts.MinLevel; level <= opts.MaxLevel; level += opts.LevelRange {
		levelCh <- level
	}
	close(levelCh)
	wg.Wait()
}

func processLevelRange(httpClient *HTTPClient, store keystore.KeyValueStore, baseURL string, opts MarketOptions, level int) {
	params := CopyParams(opts.URLListItems.Params)
	if opts.RecentItems {
		params["order"] = "desc"
		params["order_col"] = "date"
	}
	params["min_level"] = strconv.Itoa(level)
	params["max_level"] = strconv.Itoa(level + opts.LevelRange)

	for page := 1; page <= opts.MaxPages; page++ {
		time.Sleep(time.Duration(1+rand.Intn(2)) * time.Second)
		params["page"] = strconv.Itoa(page)
		url := models.ListItemsURL{Url: opts.URLListItems.Url, Params: params}.String()

		body, err := httpClient.Do("GET", url)
		if err != nil {
			log.Printf("Error fetching level %d, page %d: %v", level, page, err)
			continue
		}

		html := string(body)
		if CheckTooQuickErrorPage(html) {
			log.Printf("Rate limited: increase wait time between calls")
			continue
		}

		items := parseMarketPage(store, html)
		if len(items) == 0 {
			log.Printf("No data for level %d-%d, page %d. Skipping remaining pages.", level, level+opts.LevelRange, page)
			break
		}

		logItems(items, page, opts.ShowAll)
		buyProfitableItems(httpClient, baseURL, items)
	}
}

func parseMarketPage(store keystore.KeyValueStore, body string) []models.MarketItem {
	idObjects := ExtractIdObject(body)
	idItems := ExtractIdItems(body)
	levels := ExtractLevels(body)
	golds := ExtractGoldAmounts(body)
	rarities := ExtractRarity(body)
	types := ExtractTypeObject(body)

	n := smallestLen(idObjects, idItems, levels, golds, rarities, types)
	if n == 0 {
		return nil
	}

	items := make([]models.MarketItem, 0, n)
	for i := 0; i < n; i++ {
		valueStr, _, err := store.Get(idObjects[i])
		if err != nil {
			log.Printf("Error reading item %s from store: %v", idObjects[i], err)
			continue
		}
		value, _ := strconv.ParseFloat(valueStr, 64)
		goldNum, err := strconv.Atoi(golds[i])
		if err != nil {
			log.Printf("Invalid gold amount for item %s: %v", idItems[i], err)
			continue
		}
		items = append(items, models.MarketItem{
			ID: idItems[i], IDObject: idObjects[i],
			Level: levels[i], Gold: float64(goldNum), Value: value,
			Rarity: rarities[i], Type: types[i],
		})
	}
	return items
}

func buyProfitableItems(httpClient *HTTPClient, baseURL string, items []models.MarketItem) {
	for _, item := range items {
		if item.Diff() <= ProfitThresholdBuy {
			continue
		}
		body, err := httpClient.Do("POST", fmt.Sprintf("%s/api/market/buy/%s", baseURL, item.ID))
		if err != nil {
			log.Printf("Error buying item %s: %v", item.ID, err)
			continue
		}
		if strings.Contains(string(body), "Something went wrong") {
			log.Printf("\033[91mInsufficient gold to buy item: %s (required: %v)\033[0m", item.ID, item.Gold)
		} else {
			log.Printf("Item bought successfully: %s --> profit: %v", item.ID, item.Diff())
		}
	}
}

func logItems(items []models.MarketItem, page int, showAll bool) {
	for _, item := range items {
		diff := item.Diff()
		switch {
		case diff > ProfitThresholdBuy:
			log.Printf("\033[96m Page: %v, %s \033[0m\n", page, item.String())
		case diff >= ProfitThresholdHighlight:
			log.Printf("\033[93m Page: %v, %s \033[0m\n", page, item.String())
		case diff >= ProfitThresholdShow || showAll:
			log.Printf(" Page: %v, %s \n", page, item.String())
		case item.Rarity == "Celestial" && diff > CelestialMaxLoss:
			log.Printf("\033[95m Page: %v, %s !!! Opportunity !!! \033[0m\n", page, item.String())
		}
	}
}

func inspectItemValue(httpClient *HTTPClient, baseURL, id string) (float64, error) {
	body, err := httpClient.Do("GET", fmt.Sprintf("%s/item/inspect/%s", baseURL, id))
	if err != nil {
		return 0, fmt.Errorf("inspecting item %s: %w", id, err)
	}
	return ExtractInspectValue(string(body)), nil
}

func smallestLen(slices ...[]string) int {
	if len(slices) == 0 {
		return 0
	}
	m := len(slices[0])
	for _, s := range slices[1:] {
		if len(s) < m {
			m = len(s)
		}
	}
	return m
}
