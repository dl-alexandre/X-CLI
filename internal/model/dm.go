package model

type DMConversation struct {
	ID          string `json:"id"`
	Title       string `json:"title,omitempty"`
	Preview     string `json:"preview,omitempty"`
	URL         string `json:"url,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	Unread      bool   `json:"unread,omitempty"`
	Participant string `json:"participant,omitempty"`
}

type DMInbox struct {
	Conversations []DMConversation `json:"conversations"`
}

type DirectMessage struct {
	Text      string `json:"text"`
	CreatedAt string `json:"created_at,omitempty"`
	Sender    string `json:"sender,omitempty"`
	Outgoing  bool   `json:"outgoing,omitempty"`
}

type DMThread struct {
	Conversation DMConversation  `json:"conversation"`
	Messages     []DirectMessage `json:"messages"`
}
