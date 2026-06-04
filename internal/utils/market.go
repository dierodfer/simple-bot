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
	reIDObject   = regexp.MustCompile(`onclick="[^"]*retrieveItem\((\d+),`)
	reIDItems    = regexp.MustCompile(`id="listing-(\d+)"`)
	reRarity     = regexp.MustCompile(`<span class="[^"]*?-item[^"]*?">([^<]+)</span>`)
	reTypeObject = regexp.MustCompile(`<span[^>]*class="[^"]*-item border-0[^"]*"[^>]*>[^<]*</span>\s*([A-Za-z]+)`)
	reGold       = regexp.MustCompile(`<td[^>]*>\s*<div[^>]*>\s*<img[^>]*src=['"]` + `/img/icons/I_GoldCoin\.png['"][^>]*>\s*([\d,]*)`)
	reInspect    = regexp.MustCompile(`(?i)<div[^>]*>\s*Value\s*</div>\s*<div[^>]*>\s*([\d,]+)\s*</div>`)
	reTooQuick   = regexp.MustCompile(`<p class="[^"]*">\s*You are doing this too quickly\. Please wait a short while before doing it again\.\s*</p>`)
)

// ExtractLevels parses item level numbers from a market listings HTML page.
func ExtractLevels(body string) []string {
	return extractRegex(body, reLevels)
}

// ExtractIDObject parses retrievable item object IDs from onclick attributes in HTML.
func ExtractIDObject(body string) []string {
	return extractRegex(body, reIDObject)
}

// ExtractIDItems parses listing IDs from HTML element IDs.
func ExtractIDItems(body string) []string {
	return extractRegex(body, reIDItems)
}

// ExtractRarity parses item rarity labels from HTML span elements.
func ExtractRarity(body string) []string {
	return extractRegex(body, reRarity)
}

// ExtractTypeObject parses item type labels from HTML span elements.
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

// ExtractGoldAmounts parses and sanitizes gold cost values from HTML table cells.
func ExtractGoldAmounts(body string) []string {
	amounts := extractRegex(body, reGold)
	for i, amount := range amounts {
		amounts[i] = sanitizeNumber(amount)
	}
	return amounts
}

// ExtractInspectValue parses the displayed item value from an inspect HTML page.
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

// CheckTooQuickErrorPage reports whether the HTML body contains the rate-limit error message.
func CheckTooQuickErrorPage(body string) bool {
	return reTooQuick.MatchString(body)
}

// CopyParams returns a shallow copy of the given query parameter map.
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
