package model

type NewsItem struct {
	Headline string `json:"headline"`
	Source   string `json:"source,omitempty"`
	Meta     string `json:"meta,omitempty"`
	URL      string `json:"url,omitempty"`
}
