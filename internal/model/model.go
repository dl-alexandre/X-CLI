package model

type TweetMetrics struct {
	Likes     int `json:"likes"`
	Retweets  int `json:"retweets"`
	Replies   int `json:"replies"`
	Quotes    int `json:"quotes"`
	Bookmarks int `json:"bookmarks"`
	Views     int `json:"views"`
}

type Author struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ScreenName      string `json:"screen_name"`
	ProfileImageURL string `json:"profile_image_url,omitempty"`
	Verified        bool   `json:"verified,omitempty"`
}

type Tweet struct {
	ID          string       `json:"id"`
	Text        string       `json:"text"`
	CreatedAt   string       `json:"created_at,omitempty"`
	Lang        string       `json:"lang,omitempty"`
	URLs        []string     `json:"urls,omitempty"`
	Author      Author       `json:"author"`
	Metrics     TweetMetrics `json:"metrics"`
	IsRetweet   bool         `json:"is_retweet,omitempty"`
	RetweetedBy string       `json:"retweeted_by,omitempty"`
}

type UserProfile struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ScreenName     string `json:"screen_name"`
	Bio            string `json:"bio,omitempty"`
	Location       string `json:"location,omitempty"`
	URL            string `json:"url,omitempty"`
	FollowersCount int    `json:"followers_count"`
	FollowingCount int    `json:"following_count"`
	TweetsCount    int    `json:"tweets_count"`
	LikesCount     int    `json:"likes_count"`
	Verified       bool   `json:"verified,omitempty"`
}

type TimelineResult struct {
	Tweets []Tweet `json:"tweets"`
	Source string  `json:"source,omitempty"`
}

type TweetThread struct {
	Tweet   Tweet   `json:"tweet"`
	Replies []Tweet `json:"replies,omitempty"`
}

type ProjectStatus struct {
	Name         string   `json:"name"`
	Binary       string   `json:"binary"`
	Version      string   `json:"version"`
	Module       string   `json:"module"`
	Implemented  []string `json:"implemented"`
	Planned      []string `json:"planned"`
	Capabilities []string `json:"capabilities"`
}

type ActionResult struct {
	Action  string `json:"action"`
	Target  string `json:"target,omitempty"`
	Success bool   `json:"success"`
	URL     string `json:"url,omitempty"`
	Message string `json:"message,omitempty"`
}

type DoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

type DoctorReport struct {
	Name      string        `json:"name"`
	Checks    []DoctorCheck `json:"checks"`
	Transport string        `json:"transport"`
	Auth      string        `json:"auth"`
}

type SaltSample struct {
	Operation   string `json:"operation"`
	TimestampMS int64  `json:"timestamp_ms"`
	Salt        string `json:"salt"`
	FullTxID    string `json:"txid"`
}

type SaltComparison struct {
	SampleA    SaltSample `json:"sample_a"`
	SampleB    SaltSample `json:"sample_b"`
	SaltMatch  bool       `json:"salt_match"`
	TimeGapSec int64      `json:"time_gap_sec"`
	SameOp     bool       `json:"same_operation"`
}
