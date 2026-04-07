package model

type List struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	URL            string `json:"url,omitempty"`
	MembersCount   int    `json:"members_count,omitempty"`
	FollowersCount int    `json:"followers_count,omitempty"`
}
