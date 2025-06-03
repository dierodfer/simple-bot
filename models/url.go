package models

import (
	"net/url"
	"strings"
)

type ListItemsURL struct {
	Url    string
	Params map[string]string
}

func (l ListItemsURL) String() string {
	if len(l.Params) == 0 {
		return l.Url
	}
	var sb strings.Builder
	sb.WriteString(l.Url)
	sb.WriteString("?")
	i := 0
	for k, v := range l.Params {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(url.QueryEscape(k))
		sb.WriteString("=")
		sb.WriteString(url.QueryEscape(v))
		i++
	}
	return sb.String()
}
