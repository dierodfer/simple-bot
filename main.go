package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	keystore "simple-bot/database"
	"simple-bot/models"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type CurlRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Cookies string
}

type levelGoldRatio struct {
	id       string
	idObject string
	level    string
	gold     float64
	ratio    float64
	value    float64
}

// Diff returns the difference between value and gold.
func (lgr *levelGoldRatio) Diff() float64 {
	return lgr.value - lgr.gold
}

var (
	start        time.Time
	reqData      *CurlRequest
	store        *keystore.Store
	urlListItems models.ListItemsURL
)

func init() {
	start = time.Now()
	// Load .env file
	_ = godotenv.Load()

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		log.Fatal("APP_BASE_URL not set in .env file")
	}

	urlListItems = models.ListItemsURL{
		Url: baseURL + "/market/listings",
		Params: map[string]string{
			//"rarity[0]": "Elite",
			//"rarity[1]": "Epic",
			//"rarity[2]": "Legendary",
			//"rarity[3]": "Celestial",
			//"rarity[4]": "Exotic",
			"type[0]": "Armour",
			"type[1]": "Shield",
			"type[2]": "Weapon",
			"type[3]": "Helmet",
			"type[4]": "Gauntlet",
			"type[5]": "Amulet",
			"type[6]": "Boots",
			"type[7]": "Greaves",
			//"min_level": "400",
			"order_col": "cost",
			"order":     "asc",
			//"page":      "1",
		},
	}

	var err error
	reqData, err = parseCurlFile("call.txt")
	if err != nil {
		log.Fatalf("Error leyendo curl: %v", err)
	}

	// Initialize the database
	store, err = keystore.NewStore("data.db")
	if err != nil {
		log.Fatal("Error to init database:", err)
	}
}

func main() {
	defer store.Close()
	fmt.Println("Iniciando análisis de mercado...")

	// Recent items
	log.Printf("Analizando artículos recientes...")
	analyzeMarket(1, 0, 4000, 500, 15, true, false)
	// All items in deep
	log.Printf("Analizando mercado en profundidad...")
	analyzeMarket(1, 50, 3500, 50, 3, false, false)
	//analyzeInspectParallel(15, 121752, 200000)

	elapsed := time.Since(start)
	fmt.Printf("Execution Time: %.3f seconds\n", elapsed.Seconds())
}

// Analyze one by one the inspecting of items begin by 1 until 10,000, get the values and save in keystore
func analyzeInspectParallel(threads int, startId int, endId int) {
	idCh := make(chan int, threads)
	doneCh := make(chan struct{}, threads)

	// Workers
	for w := 0; w < threads; w++ {
		go func() {
			for i := range idCh {
				id := strconv.Itoa(i)
				//fmt.Printf("Inspecting item: %s\n", id)
				value := inspectItemValue(id)
				if value > 0 {
					fmt.Printf("Item %s value: %.2f\n", id, value)
					err := store.Set(id, fmt.Sprintf("%.0f", value))
					if err != nil {
						log.Printf("Error saving item %s: %v", id, err)
					}
				}
				time.Sleep(100 * time.Millisecond) // To avoid rate limiting
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

func analyzeMarket(threads int, minLevel int, maxLevel int, levelRange int, maxPages int, recentItems bool, showAll bool) {
	levelCh := make(chan int, threads)
	doneCh := make(chan struct{}, threads)

	for i := 0; i < threads; i++ {
		go func() {
			for level := range levelCh {
				time.Sleep(time.Duration(1+rand.Intn(10)) * time.Second)
				//fmt.Printf("Analizing levels %v until %v ...\n", level, level+levelRange)

				//Avoid concurrency between threads
				params := copyParams()
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
					//fmt.Printf("Analizing page %v level: %v-%v ...\n", page, level, level+levelRange)
					body, err := callGetMethod(reqData, url)
					if err != nil {
						log.Fatalf("Error haciendo petición para nivel %d, página %d: %v", level, page, err)
					}

					bodyString := string(body)

					if checkTooQuickErrorPage(bodyString) {
						log.Printf("Error Page detected: Please increise time to wait between calls.")
						continue
					}

					levels := extractLevels(bodyString)
					golds := extractGoldAmounts(bodyString)
					idObjects := extractIdItemsGeneric(bodyString)
					idItems := extractIdItems(bodyString)

					// Check if the lengths are 0
					if len(levels) == 0 || len(golds) == 0 || len(idObjects) == 0 || len(idItems) == 0 {
						log.Printf("Warning: No data found for level %d-%d, page %d.", level, level+levelRange, page)
						continue
					}

					listItemsOrdered := calculateDiffGold(idObjects, idItems, levels, golds)
					showItems(listItemsOrdered, page, showAll)
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

func checkTooQuickErrorPage(body string) bool {
	regex := `<p class="[^"]*">\s*You are doing this too quickly\. Please wait a short while before doing it again\.\s*</p>`
	matched, _ := regexp.MatchString(regex, body)
	return matched
}

func copyParams() map[string]string {
	// Copia local de los parámetros para evitar condiciones de carrera
	params := make(map[string]string)
	for k, v := range urlListItems.Params {
		params[k] = v
	}
	return params
}

// Get the id of an object and retrieve the value from database, then calculate the difference with the gold amount
func calculateDiffGold(idObjects []string, idItems []string, levels []string, goldAmounts []string) []levelGoldRatio {
	var itemList []levelGoldRatio
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
		itemList = append(itemList, levelGoldRatio{idItems[i], idObjects[i], levels[i], float64(goldNum), 0, value})
	}

	////Sort by diff descending
	//sort.Slice(itemList, func(i, j int) bool {
	//	return itemList[i].Diff() > itemList[j].Diff()
	//})

	return itemList
}

func showItems(itemList []levelGoldRatio, page int, showAll bool) {
	for _, lgr := range itemList {
		diff := lgr.Diff()
		if diff >= 10000 {
			fmt.Printf("\033[33m Page: %v, Level %s => %.0f 🪙 | value: %.0f | diff: %.0f | id: %s \033[0m\n", page, lgr.level, lgr.gold, lgr.value, diff, lgr.id)
		} else if diff >= 5000 {
			fmt.Printf("\033[32m Page: %v, Level %s => %.0f 🪙 | value: %.0f | diff: %.0f | id: %s \033[0m\n", page, lgr.level, lgr.gold, lgr.value, diff, lgr.id)
		} else if diff >= 1000 {
			fmt.Printf(" Page: %v, Level %s => %.0f 🪙 | value: %.0f | diff: %.0f | id: %s\n", page, lgr.level, lgr.gold, lgr.value, diff, lgr.id)
		} else if showAll {
			fmt.Printf(" Page: %v, Level %s => %.0f 🪙 | value: %.0f | diff: %.0f | id: %s\n", page, lgr.level, lgr.gold, lgr.value, diff, lgr.id)
		}
	}
}
func inspectItemValue(idGeneric string) float64 {
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		log.Fatal("APP_BASE_URL not set in .env file")
	}
	url := fmt.Sprintf("%s/item/inspect/%s", baseURL, idGeneric)
	body, err := callGetMethod(reqData, url)
	if err != nil {
		log.Fatalf("Error haciendo petición: %v", err)
	}

	return extractInspectValue(string(body))
}

func extractIdItemsGeneric(body string) []string {
	re := regexp.MustCompile(`onclick="[^"]*retrieveItem\((\d+),`)
	matches := re.FindAllStringSubmatch(body, -1)
	var ids []string
	for _, match := range matches {
		if len(match) > 1 {
			ids = append(ids, match[1])
		}
	}

	return ids
}

func extractIdItems(body string) []string {
	re := regexp.MustCompile(`id="listing-(\d+)"`)
	matches := re.FindAllStringSubmatch(body, -1)
	var ids []string
	for _, match := range matches {
		if len(match) > 1 {
			ids = append(ids, match[1])
		}
	}

	return ids
}

func calculateRatios(idItems []string, idItemGeneric []string, levels []string, goldAmounts []string) []levelGoldRatio {
	if len(levels) != len(goldAmounts) || len(levels) != len(idItemGeneric) || len(levels) != len(idItems) {
		log.Printf("Warning: levels, goldAmounts, idItemGeneric, and idItems have different lengths (levels: %d, goldAmounts: %d, idItemGeneric: %d, idItems: %d)", len(levels), len(goldAmounts), len(idItemGeneric), len(idItems))
	}

	var lgrList []levelGoldRatio
	for i := range levels {
		levelNum, _ := strconv.Atoi(levels[i])
		goldNum, _ := strconv.Atoi(goldAmounts[i])
		ratio := 0.0
		if goldNum != 0 {
			ratio = float64(levelNum) / float64(goldNum)
		}
		lgrList = append(lgrList, levelGoldRatio{idItems[i], idItemGeneric[i], levels[i], float64(goldNum), ratio, 0.0})
	}
	// Sort by ratio descending
	sort.Slice(lgrList, func(i, j int) bool {
		return lgrList[i].ratio > lgrList[j].ratio
	})

	return lgrList
}

func extractGoldAmounts(body string) []string {
	goldRegex := regexp.MustCompile(`<td[^>]*>\s*<div[^>]*>\s*<img[^>]*src=['"]/img/icons/I_GoldCoin\.png['"][^>]*>\s*([\d,]*)`)
	matches := goldRegex.FindAllStringSubmatch(body, -1)

	var goldAmounts []string
	for _, m := range matches {
		amount := strings.ReplaceAll(m[1], ",", "")
		if amount == "" {
			amount = "0" // o ignora según lo que necesites
		}
		goldAmounts = append(goldAmounts, amount)
	}
	return goldAmounts
}

func extractInspectValue(body string) float64 {
	re := regexp.MustCompile(`(?i)<div[^>]*>\s*Value\s*</div>\s*<div[^>]*>\s*([\d,]+)\s*</div>`)
	match := re.FindStringSubmatch(body)
	if len(match) > 1 {
		valueStr := strings.ReplaceAll(match[1], ",", "")
		value, err := strconv.ParseFloat(valueStr, 64)
		if err == nil {
			return value
		}
		log.Println("Error parsing value:", err)
	} else {
		log.Println("Valor no encontrado")
	}
	return 0
}

func extractLevels(body string) []string {
	levelRegex := regexp.MustCompile(`Level (\d{1,4})`)
	matches := levelRegex.FindAllStringSubmatch(body, -1)
	var levels []string
	for _, m := range matches {
		levels = append(levels, m[1])
	}
	return levels
}

func parseCurlFile(path string) (*CurlRequest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Unir las líneas para reconstruir el comando
	content := strings.Join(lines, " ")
	content = strings.ReplaceAll(content, "^", "")
	content = strings.TrimSpace(content)

	//curlRegex := regexp.MustCompile(`curl\s+['"]?([^'"\\ ]+)['"]?`)
	//match := curlRegex.FindStringSubmatch(content)
	//if len(match) < 2 {
	//	return nil, fmt.Errorf("URL no encontrada")
	//}
	//url := match[1]

	headers := map[string]string{}
	headerRegex := regexp.MustCompile(`-H\s+['"]([^:]+):\s?(.+?)['"]`)
	for _, h := range headerRegex.FindAllStringSubmatch(content, -1) {
		headers[h[1]] = h[2]
	}

	cookieRegex := regexp.MustCompile(`-b\s+['"](.+?)['"]`)
	cookie := ""
	if match := cookieRegex.FindStringSubmatch(content); len(match) == 2 {
		cookie = match[1]
	}

	method := "GET"
	if strings.Contains(content, "-X") {
		methodRegex := regexp.MustCompile(`-X\s+['"]?(\w+)['"]?`)
		if match := methodRegex.FindStringSubmatch(content); len(match) > 1 {
			method = strings.ToUpper(match[1])
		}
	}

	return &CurlRequest{
		Method: method,
		//URL:     url,
		Headers: headers,
		Cookies: cookie,
	}, nil
}

func callGetMethod(reqData *CurlRequest, url string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest(reqData.Method, url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range reqData.Headers {
		req.Header.Set(k, v)
	}
	if reqData.Cookies != "" {
		req.Header.Set("Cookie", reqData.Cookies)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
