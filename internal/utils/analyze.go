package utils

import (
	"fmt"
	"log"
	"math/rand"
	config "simple-bot/configs"
	keystore "simple-bot/internal/database"
	"simple-bot/internal/models"
	"strconv"
	"strings"
	"time"
)

func AnalyzeInspectParallel(threads int, startId int, endId int) {
	idCh := make(chan int, threads)
	doneCh := make(chan struct{}, threads)

	for w := 0; w < threads; w++ {
		go func() {
			for i := range idCh {
				id := strconv.Itoa(i)
				value := InspectItemValue(id)
				if value > 0 {
					log.Printf("Item %s value: %.2f\n", id, value)
					err := keystore.Database.Set(id, fmt.Sprintf("%.0f", value))
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

func AnalyzeMarket(urlListItems models.ListItemsURL, threads int, minLevel int, maxLevel int, levelRange int, maxPages int, recentItems bool, showAll bool) {
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
					body, err := HttpCall("GET", url)
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
					rarities := ExtractRarity(bodyString)
					typeObjects := ExtractTypeObject(bodyString)
					if len(typeObjects) == 0 || len(levels) == 0 || len(golds) == 0 || len(idObjects) == 0 || len(idItems) == 0 || len(rarities) == 0 {
						log.Printf("Warning: No data found for level %d-%d, page %d. Skipping next pages...", level, level+levelRange, page)
						break
					}
					listItemsOrdered := CalculateDiffGold(idObjects, idItems, levels, golds, rarities, typeObjects)
					showItemsByDiff(listItemsOrdered, page, showAll)
					buyAndSellItems(listItemsOrdered)
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

func buyAndSellItems(itemList []models.MarketItem) {
	for _, item := range itemList {
		//levelInt, _ := strconv.Atoi(item.Level)
		if item.Diff() > 10000 {
			url := fmt.Sprintf("%s/api/market/buy/%s", config.BaseURL, item.ID)
			body, err := HttpCall("POST", url)
			if err != nil {
				log.Printf("Error buying item %s: %v", item.ID, err)
			}
			if strings.Contains(string(body), "Something went wrong") {
				log.Printf("Insufficient gold to buy item: %s (required: %v 🪙)", item.ID, item.Gold)
			} else {
				log.Printf("Item bought successfully: %s --> profit: %v", item.String(), item.Diff())
			}

			//time.Sleep(time.Duration(1+rand.Intn(10)) * time.Second)
			//url = fmt.Sprintf("%s/quicksell/item/%s", config.BaseURL, item.IDObject)
			//_, err = HttpCall("POST", url)
			//if err != nil {
			//	log.Printf("Error selling item %s: %v", item.IDObject, err)
			//}
			//log.Printf("Sold item: %s gold earned: %.0f", item.String(), item.Diff())
		}
	}
}

func CalculateDiffGold(idObjects []string, idItems []string, levels []string, goldAmounts []string, rarities []string, typeObjects []string) []models.MarketItem {
	var itemList []models.MarketItem
	if len(idObjects) != len(goldAmounts) {
		log.Printf("Alert: idObject and golds have different lengths (idObject: %d, golds: %d)", len(idObjects), len(goldAmounts))
		return itemList
	}

	for i := range idObjects {
		id := idObjects[i]
		valueStr, found, _ := keystore.Database.Get(id)
		if !found {
			log.Printf("Item %s not found in database", id)
		}
		value, _ := strconv.ParseFloat(valueStr, 64)
		goldNum, _ := strconv.Atoi(goldAmounts[i])
		itemList = append(itemList, models.MarketItem{ID: idItems[i], IDObject: idObjects[i], Level: levels[i], Gold: float64(goldNum), Value: value, Rarity: rarities[i], Type: typeObjects[i]})
	}
	return itemList
}

func showItemsByDiff(itemList []models.MarketItem, page int, showAll bool) {
	for _, item := range itemList {
		diff := item.Diff()
		if diff > 15000 {
			log.Printf("\033[96m Page: %v, %s \033[0m\n", page, item.String())
		} else if diff >= 10000 {
			log.Printf("\033[93m Page: %v, %s \033[0m\n", page, item.String())
		} else if diff >= 5000 {
			log.Printf("\033[92m Page: %v, %s \033[0m\n", page, item.String())
		} else if diff >= 1000 {
			log.Printf(" Page: %v, %s \n", page, item.String())
		} else if showAll {
			log.Printf(" Page: %v, %s \n", page, item.String())
		} else if item.Rarity == "Celestial" && diff > -300000 {
			log.Printf("\033[95m Page: %v, %s !!! Oportunity ¡¡¡ \033[0m\n", page, item.String())
		}
	}
}

func InspectItemValue(idGeneric string) float64 {

	url := fmt.Sprintf("%s/item/inspect/%s", config.BaseURL, idGeneric)
	body, err := HttpCall("GET", url)
	if err != nil {
		log.Fatalf("Error haciendo petición: %v", err)
	}
	return ExtractInspectValue(string(body))
}
