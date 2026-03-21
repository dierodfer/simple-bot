package utils

import (
	"regexp"
	"strconv"
	"strings"
)

// Pre-compiled regular expressions for HTML scraping.
// Compiled once at package init to avoid repeated compilation overhead.
var (
	reLevels     = regexp.MustCompile(`Level (\d{1,4})`)
	reIdObject   = regexp.MustCompile(`onclick="[^"]*retrieveItem\((\d+),`)
	reIdItems    = regexp.MustCompile(`id="listing-(\d+)"`)
	reRarity     = regexp.MustCompile(`<span class="[^"]*?-item[^"]*?">([^<]+)</span>`)
	reTypeObject = regexp.MustCompile(`<span[^>]*class="[^"]*-item border-0[^"]*"[^>]*>[^<]*</span>\s*([A-Za-z]+)`)
	reGold       = regexp.MustCompile(`<td[^>]*>\s*<div[^>]*>\s*<img[^>]*src=['"]/img/icons/I_GoldCoin\.png['"][^>]*>\s*([\d,]*)`)
	reInspect    = regexp.MustCompile(`(?i)<div[^>]*>\s*Value\s*</div>\s*<div[^>]*>\s*([\d,]+)\s*</div>`)
	reTooQuick   = regexp.MustCompile(`<p class="[^"]*">\s*You are doing this too quickly\. Please wait a short while before doing it again\.\s*</p>`)
)

func ExtractLevels(body string) []string {
	return extractRegex(body, reLevels)
}

func ExtractIdObject(body string) []string {
	return extractRegex(body, reIdObject)
}

func ExtractIdItems(body string) []string {
	return extractRegex(body, reIdItems)
}

func ExtractRarity(body string) []string {
	return extractRegex(body, reRarity)
}

func ExtractTypeObject(body string) []string {
	return extractRegex(body, reTypeObject)
}

func extractRegex(body string, re *regexp.Regexp) []string {
	matches := re.FindAllStringSubmatch(body, -1)
	elements := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 {
			elements = append(elements, m[1])
		}
	}
	return elements
}

func ExtractGoldAmounts(body string) []string {
	amounts := extractRegex(body, reGold)
	for i, amount := range amounts {
		amounts[i] = sanitizeNumber(amount)
	}
	return amounts
}

func ExtractInspectValue(body string) float64 {
	match := reInspect.FindStringSubmatch(body)
	if len(match) <= 1 {
		return 0
	}
	value, err := strconv.ParseFloat(sanitizeNumber(match[1]), 64)
	if err != nil {
		return 0
	}
	return value
}

func CheckTooQuickErrorPage(body string) bool {
	return reTooQuick.MatchString(body)
}

func CopyParams(params map[string]string) map[string]string {
	cloned := make(map[string]string, len(params))
	for k, v := range params {
		cloned[k] = v
	}
	return cloned
}

func sanitizeNumber(value string) string {
	return strings.ReplaceAll(value, ",", "")
}
