package models

type CurlRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Cookies string
}
