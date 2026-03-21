package models

// CurlRequest represents the parsed components of a cURL command.
type CurlRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Cookies string
}
