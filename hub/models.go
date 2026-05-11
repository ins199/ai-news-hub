package hub

import "time"

type Article struct {
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Summary     string    `json:"summary"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`
	ImageURL    string    `json:"image_url,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
}

type FetchFunc func() ([]Article, error)

type SourceConfig struct {
	Name  string
	Fetch FetchFunc
}

var Sources = []SourceConfig{
	{Name: "IT之家", Fetch: FetchITHomeRSS},
	{Name: "爱范儿", Fetch: FetchIfanr},
	{Name: "少数派", Fetch: FetchSSPai},
	{Name: "Hacker News", Fetch: FetchHN},
}
