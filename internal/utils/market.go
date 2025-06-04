package utils

import (
	"regexp"
	"strconv"
	"strings"
)

func ExtractIdItemsGeneric(body string) []string {
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

func ExtractIdItems(body string) []string {
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

func ExtractGoldAmounts(body string) []string {
	goldRegex := regexp.MustCompile(`<td[^>]*>\s*<div[^>]*>\s*<img[^>]*src=['"]/img/icons/I_GoldCoin\.png['"][^>]*>\s*([\d,]*)`)
	matches := goldRegex.FindAllStringSubmatch(body, -1)

	var goldAmounts []string
	for _, m := range matches {
		amount := strings.ReplaceAll(m[1], ",", "")
		if amount == "" {
			amount = "0"
		}
		goldAmounts = append(goldAmounts, amount)
	}
	return goldAmounts
}

func ExtractLevels(body string) []string {
	levelRegex := regexp.MustCompile(`Level (\d{1,4})`)
	matches := levelRegex.FindAllStringSubmatch(body, -1)
	var levels []string
	for _, m := range matches {
		levels = append(levels, m[1])
	}
	return levels
}

func ExtractInspectValue(body string) float64 {
	re := regexp.MustCompile(`(?i)<div[^>]*>\s*Value\s*</div>\s*<div[^>]*>\s*([\d,]+)\s*</div>`)
	match := re.FindStringSubmatch(body)
	if len(match) > 1 {
		valueStr := strings.ReplaceAll(match[1], ",", "")
		value, err := strconv.ParseFloat(valueStr, 64)
		if err == nil {
			return value
		}
	}
	return 0
}

func CheckTooQuickErrorPage(body string) bool {
	regex := `<p class="[^"]*">\s*You are doing this too quickly\. Please wait a short while before doing it again\.\s*</p>`
	matched, _ := regexp.MatchString(regex, body)
	return matched
}

func CopyParams(params map[string]string) map[string]string {
	copyVars := make(map[string]string)
	for k, v := range params {
		copyVars[k] = v
	}
	return copyVars
}
