package model

type Community struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	URL          string `json:"url,omitempty"`
	MembersCount int    `json:"members_count,omitempty"`
	RulesCount   int    `json:"rules_count,omitempty"`
}
