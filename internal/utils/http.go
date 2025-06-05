package utils

import (
	"bufio"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var Headers map[string]string
var Cookie string

// HttpCall executes an HTTP request using the CurlRequest model.
func HttpCall(method string, url string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range Headers {
		req.Header.Set(k, v)
	}

	req.Header.Set("Cookie", Cookie)

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

// InitHeadersAndCookie initializes the Headers and Cookie variables from a file containing curl command options.
func InitHeadersAndCookie(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
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

	//method := "GET"
	//if strings.Contains(content, "-X") {
	//	methodRegex := regexp.MustCompile(`-X\s+['"]?(\w+)['"]?`)
	//	if match := methodRegex.FindStringSubmatch(content); len(match) > 1 {
	//		method = strings.ToUpper(match[1])
	//	}
	//}

	Headers = headers
	Cookie = cookie

	return nil
}
