package model

type Trend struct {
	Name     string `json:"name"`
	Category string `json:"category,omitempty"`
	Posts    string `json:"posts,omitempty"`
	URL      string `json:"url,omitempty"`
}
