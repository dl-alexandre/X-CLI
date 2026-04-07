package model

type Space struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Host        string `json:"host,omitempty"`
	State       string `json:"state,omitempty"`
	ScheduledAt string `json:"scheduled_at,omitempty"`
	URL         string `json:"url,omitempty"`
}
