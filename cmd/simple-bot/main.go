package main

import (
	"fmt"
	"log"
	config "simple-bot/configs"
	keystore "simple-bot/internal/database"
	"simple-bot/internal/models"
	"simple-bot/internal/utils"
	"time"
)

var (
	start        time.Time
	urlListItems models.ListItemsURL
)

func init() {
	start = time.Now()
	config.InitVars()

	urlListItems = models.ListItemsURL{
		Url: config.BaseURL + "/market/listings",
		Params: map[string]string{
			"type[0]":   "Armour",
			"type[1]":   "Shield",
			"type[2]":   "Weapon",
			"type[3]":   "Helmet",
			"type[4]":   "Gauntlet",
			"type[5]":   "Amulet",
			"type[6]":   "Boots",
			"type[7]":   "Greaves",
			"order_col": "cost",
			"order":     "asc",
		},
	}

	err := utils.InitHeadersAndCookie("call.txt")
	if err != nil {
		log.Fatalf("Error leyendo curl: %v", err)
	}

	err = keystore.NewStore("internal/database/data.db")
	if err != nil {
		log.Fatal("Error to init database:", err)
	}
}

func main() {
	fmt.Println("Iniciando base de datos...")
	defer keystore.Database.Close()

	log.Printf("Analizando artículos recientes...")
	utils.AnalyzeMarket(urlListItems, 1, 0, 5500, 500, 20, true, false)
	log.Printf("Analizando mercado en profundidad...")
	utils.AnalyzeMarket(urlListItems, 1, 100, 5700, 100, 1, false, false)
	//utils.AnalyzeInspectParallel(1, 0, 1)

	//url := fmt.Sprintf("%s/item/inspect/%s", baseURL, idGeneric)

	elapsed := time.Since(start)
	fmt.Printf("Execution Time: %.3f seconds\n", elapsed.Seconds())
}
