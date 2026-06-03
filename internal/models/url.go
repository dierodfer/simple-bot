package models

import (
	"net/url"
)

type ListItemsURL struct {
	Url    string
	Params map[string]string
}

func (l ListItemsURL) String() string {
	if len(l.Params) == 0 {
		return l.Url
	}
	vals := url.Values{}
	for k, v := range l.Params {
		vals.Set(k, v)
	}
	return l.Url + "?" + vals.Encode()
}
