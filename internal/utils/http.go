package utils

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const defaultHTTPTimeout = 30 * time.Second

// HTTPClient wraps HTTP configuration for making authenticated requests.
type HTTPClient struct {
	headers map[string]string
	cookie  string
	client  *http.Client
}

// NewHTTPClient creates an HTTPClient by parsing headers and cookies from a curl-style command file.
func NewHTTPClient(path string) (*HTTPClient, error) {
	headers, cookie, err := parseCurlFile(path)
	if err != nil {
		return nil, fmt.Errorf("parsing curl file: %w", err)
	}

	return &HTTPClient{
		headers: headers,
		cookie:  cookie,
		client:  &http.Client{Timeout: defaultHTTPTimeout},
	}, nil
}

// Do executes an HTTP request and returns the response body.
func (c *HTTPClient) Do(method, url string) ([]byte, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Cookie", c.cookie)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	return body, nil
}

func parseCurlFile(path string) (map[string]string, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
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

	headers := make(map[string]string)
	headerRegex := regexp.MustCompile(`-H\s+['"]([^:]+):\s?(.+?)['"]`)
	for _, h := range headerRegex.FindAllStringSubmatch(content, -1) {
		headers[h[1]] = h[2]
	}

	cookie := ""
	cookieRegex := regexp.MustCompile(`-b\s+['"](.+?)['"]`)
	if match := cookieRegex.FindStringSubmatch(content); len(match) == 2 {
		cookie = match[1]
	}

	return headers, cookie, nil
}
