package main

import (
	"fmt"
	"log"
	"os"
	keystore "simple-bot/internal/database"
	"simple-bot/internal/models"
	"simple-bot/internal/utils"
	"time"

	"github.com/joho/godotenv"
)

var (
	start        time.Time
	reqData      *models.CurlRequest
	store        *keystore.Store
	urlListItems models.ListItemsURL
)

func init() {
	start = time.Now()
	_ = godotenv.Load()

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		log.Fatal("APP_BASE_URL not set in .env file")
	}

	urlListItems = models.ListItemsURL{
		Url: baseURL + "/market/listings",
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

	var err error
	reqData, err = utils.ParseCurlFile("call.txt")
	if err != nil {
		log.Fatalf("Error leyendo curl: %v", err)
	}

	store, err = keystore.NewStore("internal/database/data.db")
	if err != nil {
		log.Fatal("Error to init database:", err)
	}
}

func main() {
	defer store.Close()
	fmt.Println("Iniciando análisis de mercado...")

	log.Printf("Analizando artículos recientes...")
	utils.AnalyzeMarket(store, reqData, urlListItems, 1, 0, 4500, 500, 15, true, false)
	log.Printf("Analizando mercado en profundidad...")
	utils.AnalyzeMarket(store, reqData, urlListItems, 1, 50, 3500, 50, 3, false, false)
	//utils.AnalyzeInspectParallel(store, 15, 121752, 200000, reqData)

	elapsed := time.Since(start)
	fmt.Printf("Execution Time: %.3f seconds\n", elapsed.Seconds())
}
