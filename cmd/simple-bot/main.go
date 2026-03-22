package main

import (
	"log"
	"os"
	"strconv"
	"time"

	config "simple-bot/configs"
	keystore "simple-bot/internal/database"
	"simple-bot/internal/models"
	"simple-bot/internal/ui"
	"simple-bot/internal/utils"
	"simple-bot/internal/version"
)

func main() {
	start := time.Now()

	if len(os.Args) < 2 {
		log.Fatalf("Simple Bot v%s\\nUsage: simple-bot <inspect|analyze|ui|version> [args...]", version.AppVersion)
	}

	if os.Args[1] == "version" || os.Args[1] == "--version" || os.Args[1] == "-v" {
		log.Printf("Simple Bot v%s", version.AppVersion)
		return
	}

	log.Printf("Simple Bot v%s", version.AppVersion)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	httpClient, err := utils.NewHTTPClient("call.txt")
	if err != nil {
		log.Fatalf("Error initializing HTTP client: %v", err)
	}

	store, err := keystore.NewStore("internal/database/data.db")
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	defer store.Close()

	urlListItems := models.ListItemsURL{
		Url: cfg.BaseURL + "/market/listings",
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

	switch os.Args[1] {
	case "inspect":
		if len(os.Args) < 4 {
			log.Fatal("Usage: simple-bot inspect <start_id> <end_id>")
		}
		initVal, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("Invalid start ID %q: %v", os.Args[2], err)
		}
		endVal, err := strconv.Atoi(os.Args[3])
		if err != nil {
			log.Fatalf("Invalid end ID %q: %v", os.Args[3], err)
		}
		log.Printf("Inspecting items in range %d-%d...", initVal, endVal)
		utils.AnalyzeInspectParallel(httpClient, store, cfg.BaseURL, 3, initVal, endVal)

	case "analyze":
		log.Println("Analyzing recent market items...")
		utils.AnalyzeMarket(httpClient, store, cfg.BaseURL, utils.MarketOptions{
			URLListItems: urlListItems,
			Threads:      1,
			MinLevel:     0,
			MaxLevel:     5500,
			LevelRange:   500,
			MaxPages:     20,
			RecentItems:  true,
			ShowAll:      false,
		})

	case "ui":
		if err := ui.Run(httpClient, store, cfg.BaseURL, utils.MarketOptions{
			URLListItems: urlListItems,
			Threads:      1,
			MinLevel:     0,
			MaxLevel:     5500,
			LevelRange:   500,
			MaxPages:     20,
			RecentItems:  true,
			ShowAll:      false,
		}); err != nil {
			log.Fatalf("UI error: %v", err)
		}

	default:
		log.Fatalf("Unknown command %q. Use 'inspect', 'analyze', 'ui', or 'version'.", os.Args[1])
	}

	log.Printf("Execution Time: %.3f seconds\n", time.Since(start).Seconds())
}
