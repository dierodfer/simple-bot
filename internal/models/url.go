package models

import (
	"net/url"
)

// ListItemsURL represents a URL with optional query parameters for the market listings API.
type ListItemsURL struct {
	URL    string
	Params map[string]string
}

// String returns the full URL string with query parameters encoded alphabetically.
func (l ListItemsURL) String() string {
	if len(l.Params) == 0 {
		return l.URL
	}
	vals := url.Values{}
	for k, v := range l.Params {
		vals.Set(k, v)
	}
	return l.URL + "?" + vals.Encode()
}
