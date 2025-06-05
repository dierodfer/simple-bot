package utils

import (
	"regexp"
	"strconv"
	"strings"
)

func ExtractLevels(body string) []string {
	re := regexp.MustCompile(`Level (\d{1,4})`)
	return extractRegex(body, re)
}

func ExtractIdObject(body string) []string {
	re := regexp.MustCompile(`onclick="[^"]*retrieveItem\((\d+),`)
	return extractRegex(body, re)
}

func ExtractIdItems(body string) []string {
	re := regexp.MustCompile(`id="listing-(\d+)"`)
	return extractRegex(body, re)
}

func ExtractRarity(body string) []string {
	re := regexp.MustCompile(`<span class="[^"]*?-item[^"]*?">([^<]+)</span>`)
	return extractRegex(body, re)
}

func ExtractTypeObject(body string) []string {
	re := regexp.MustCompile(`<span[^>]*class="[^"]*-item border-0[^"]*"[^>]*>[^<]*</span>\s*([A-Za-z]+)`)
	return extractRegex(body, re)
}

func extractRegex(body string, re *regexp.Regexp) []string {
	matches := re.FindAllStringSubmatch(body, -1)
	elements := make([]string, 0, len(matches))
	for _, m := range matches {
		elements = append(elements, m[1])
	}
	return elements
}

func ExtractGoldAmounts(body string) []string {
	re := regexp.MustCompile(`<td[^>]*>\s*<div[^>]*>\s*<img[^>]*src=['"]/img/icons/I_GoldCoin\.png['"][^>]*>\s*([\d,]*)`)
	amounts := extractRegex(body, re)
	for i, amount := range amounts {
		amounts[i] = strings.ReplaceAll(amount, ",", "")
	}
	return amounts
}

func ExtractInspectValue(body string) float64 {
	re := regexp.MustCompile(`(?i)<div[^>]*>\s*Value\s*</div>\s*<div[^>]*>\s*([\d,]+)\s*</div>`)
	match := re.FindStringSubmatch(body)
	if len(match) > 1 {
		valueStr := strings.ReplaceAll(match[1], ",", "")
		if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
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
