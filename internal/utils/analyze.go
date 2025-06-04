package utils

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	keystore "simple-bot/internal/database"
	"simple-bot/internal/models"
	"strconv"
	"time"
)

func AnalyzeInspectParallel(store *keystore.Store, threads int, startId int, endId int, reqData *models.CurlRequest) {
	idCh := make(chan int, threads)
	doneCh := make(chan struct{}, threads)

	for w := 0; w < threads; w++ {
		go func() {
			for i := range idCh {
				id := strconv.Itoa(i)
				value := InspectItemValue(reqData, id)
				if value > 0 {
					fmt.Printf("Item %s value: %.2f\n", id, value)
					err := store.Set(id, fmt.Sprintf("%.0f", value))
					if err != nil {
						log.Printf("Error saving item %s: %v", id, err)
					}
				}
				time.Sleep(100 * time.Millisecond)
			}
			doneCh <- struct{}{}
		}()
	}

	for i := startId; i <= endId; i++ {
		idCh <- i
	}
	close(idCh)

	for w := 0; w < threads; w++ {
		<-doneCh
	}
}

func AnalyzeMarket(store *keystore.Store, reqData *models.CurlRequest, urlListItems models.ListItemsURL, threads int, minLevel int, maxLevel int, levelRange int, maxPages int, recentItems bool, showAll bool) {
	levelCh := make(chan int, threads)
	doneCh := make(chan struct{}, threads)

	for i := 0; i < threads; i++ {
		go func() {
			for level := range levelCh {
				time.Sleep(time.Duration(1+rand.Intn(10)) * time.Second)
				params := CopyParams(urlListItems.Params)
				if recentItems {
					params["order"] = "desc"
					params["order_col"] = "date"
				}
				params["min_level"] = strconv.Itoa(level)
				params["max_level"] = strconv.Itoa(level + levelRange)
				for page := 1; page <= maxPages; page++ {
					time.Sleep(time.Duration(1+rand.Intn(2)) * time.Second)
					params["page"] = strconv.Itoa(page)
					url := models.ListItemsURL{
						Url:    urlListItems.Url,
						Params: params,
					}.String()
					body, err := GetMethod(reqData, url)
					if err != nil {
						log.Fatalf("Error haciendo petición para nivel %d, página %d: %v", level, page, err)
					}
					bodyString := string(body)
					if CheckTooQuickErrorPage(bodyString) {
						log.Printf("Error Page detected: Please increise time to wait between calls.")
						continue
					}
					levels := ExtractLevels(bodyString)
					golds := ExtractGoldAmounts(bodyString)
					idObjects := ExtractIdObject(bodyString)
					idItems := ExtractIdItems(bodyString)
					if len(levels) == 0 || len(golds) == 0 || len(idObjects) == 0 || len(idItems) == 0 {
						log.Printf("Warning: No data found for level %d-%d, page %d.", level, level+levelRange, page)
						continue
					}
					listItemsOrdered := CalculateDiffGold(store, idObjects, idItems, levels, golds)
					ShowItems(listItemsOrdered, page, showAll)
				}
			}
			doneCh <- struct{}{}
		}()
	}

	for level := minLevel; level <= maxLevel; level += levelRange {
		levelCh <- level
	}
	close(levelCh)

	for i := 0; i < threads; i++ {
		<-doneCh
	}
}

func CalculateDiffGold(store *keystore.Store, idObjects []string, idItems []string, levels []string, goldAmounts []string) []models.MarketItem {
	var itemList []models.MarketItem
	if len(idObjects) != len(goldAmounts) {
		log.Printf("Alert: idObject and golds have different lengths (idObject: %d, golds: %d)", len(idObjects), len(goldAmounts))
		return itemList
	}

	for i := range idObjects {
		id := idObjects[i]
		valueStr, found, _ := store.Get(id)
		if !found {
			log.Printf("Item %s not found in database", id)
		}
		value, _ := strconv.ParseFloat(valueStr, 64)
		goldNum, _ := strconv.Atoi(goldAmounts[i])
		itemList = append(itemList, models.MarketItem{ID: idItems[i], IDObject: idObjects[i], Level: levels[i], Gold: float64(goldNum), Value: value})
	}
	return itemList
}

func ShowItems(itemList []models.MarketItem, page int, showAll bool) {
	for _, lgr := range itemList {
		diff := lgr.Diff()
		if diff >= 10000 {
			fmt.Printf("\033[33m Page: %v, %s \033[0m\n", page, lgr.String())
		} else if diff >= 5000 {
			fmt.Printf("\033[32m Page: %v, %s \033[0m\n", page, lgr.String())
		} else if diff >= 1000 {
			fmt.Printf(" Page: %v, %s \n", page, lgr.String())
		} else if showAll {
			fmt.Printf(" Page: %v, %s \n", page, lgr.String())
		}
	}
}

func InspectItemValue(reqData *models.CurlRequest, idGeneric string) float64 {
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		log.Fatal("APP_BASE_URL not set in .env file")
	}
	url := fmt.Sprintf("%s/item/inspect/%s", baseURL, idGeneric)
	body, err := GetMethod(reqData, url)
	if err != nil {
		log.Fatalf("Error haciendo petición: %v", err)
	}
	return ExtractInspectValue(string(body))
}
