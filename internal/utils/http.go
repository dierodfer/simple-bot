package utils

import (
	"bufio"
	"io"
	"net/http"
	"os"
	"regexp"
	"simple-bot/internal/models"
	"strings"
)

// GetMethod executes an HTTP request using the CurlRequest model.
func GetMethod(reqData *models.CurlRequest, url string) ([]byte, error) {
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

// ParseCurlFile parses a curl command from a file into a CurlRequest struct.
func ParseCurlFile(path string) (*models.CurlRequest, error) {
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

	content := strings.Join(lines, " ")
	content = strings.ReplaceAll(content, "^", "")
	content = strings.TrimSpace(content)

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

	return &models.CurlRequest{
		Method:  method,
		Headers: headers,
		Cookies: cookie,
	}, nil
}
