package hub

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)
var whitespaceRe = regexp.MustCompile(`\s+`)
var imgRe = regexp.MustCompile(`<img[^>]+src=["']([^"']+)["']`)

// ── generic RSS fetcher ──

func fetchRSS(url string) ([]*gofeed.Item, error) {
	fp := gofeed.NewParser()
	fp.UserAgent = "ai-news-hub/1.0"
	feed, err := fp.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return feed.Items, nil
}

func rssToArticles(items []*gofeed.Item, source string) []Article {
	var articles []Article
	for _, item := range items {
		pubDate := time.Now()
		if item.PublishedParsed != nil {
			pubDate = *item.PublishedParsed
		}

		summary := cleanHTML(item.Description)
		if summary == "" {
			summary = cleanHTML(item.Content)
		}

		imgURL := extractImage(item.Description)
		if imgURL == "" {
			imgURL = extractImage(item.Content)
		}

		var tags []string
		for _, cat := range item.Categories {
			t := strings.TrimSpace(cat)
			if t != "" {
				tags = append(tags, t)
			}
		}

		articles = append(articles, Article{
			Title:       item.Title,
			Link:        item.Link,
			Summary:     summary,
			Source:      source,
			PublishedAt: pubDate,
			ImageURL:    imgURL,
			Tags:        tags,
		})
	}
	return articles
}

func extractImage(html string) string {
	m := imgRe.FindStringSubmatch(html)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func cleanHTML(raw string) string {
	s := htmlTagRe.ReplaceAllString(raw, " ")
	s = whitespaceRe.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	if len(s) > 300 {
		s = s[:300]
		if idx := strings.LastIndexAny(s, " ，。！？,.!?"); idx > 200 {
			s = s[:idx]
		}
		s += "..."
	}
	return s
}

// ── IT之家 RSS ──

func FetchITHomeRSS() ([]Article, error) {
	items, err := fetchRSS("https://www.ithome.com/rss/")
	if err != nil {
		return nil, err
	}
	articles := rssToArticles(items, "IT之家")
	if len(articles) > 15 {
		articles = articles[:15]
	}
	return articles, nil
}

// ── 爱范儿 RSS ──

func FetchIfanr() ([]Article, error) {
	items, err := fetchRSS("https://www.ifanr.com/feed")
	if err != nil {
		return nil, err
	}
	articles := rssToArticles(items, "爱范儿")
	if len(articles) > 15 {
		articles = articles[:15]
	}
	return articles, nil
}

// ── 少数派 RSS ──

func FetchSSPai() ([]Article, error) {
	items, err := fetchRSS("https://sspai.com/feed")
	if err != nil {
		return nil, err
	}
	articles := rssToArticles(items, "少数派")
	if len(articles) > 15 {
		articles = articles[:15]
	}
	return articles, nil
}

// ── Hacker News via Algolia ──

type algoliaHit struct {
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Points      int      `json:"points"`
	NumComments int      `json:"num_comments"`
	CreatedAt   string   `json:"created_at"`
	Tags        []string `json:"_tags"`
}

func FetchHN() ([]Article, error) {
	url := "https://hn.algolia.com/api/v1/search_by_date?tags=story&hitsPerPage=20"
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "ai-news-hub/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("algolia %d", resp.StatusCode)
	}
	var result struct {
		Hits []algoliaHit `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	var articles []Article
	for _, hit := range result.Hits {
		pubDate := time.Now()
		if t, err := time.Parse(time.RFC3339, hit.CreatedAt); err == nil {
			pubDate = t
		}
		articles = append(articles, Article{
			Title:       hit.Title,
			Link:        hit.URL,
			Summary:     fmt.Sprintf("%d points · %d comments", hit.Points, hit.NumComments),
			Source:      "Hacker News",
			PublishedAt: pubDate,
		})
	}
	return articles, nil
}

// RefreshAll 抓取所有源并返回合并后的文章列表
func RefreshAll() ([]Article, error) {
	var all []Article
	for _, src := range Sources {
		articles, err := src.Fetch()
		if err != nil {
			log.Printf("[WARN] %s: %v", src.Name, err)
			continue
		}
		log.Printf("[INFO] %s: %d articles", src.Name, len(articles))
		all = append(all, articles...)
	}
	// 按发布时间倒序
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].PublishedAt.After(all[i].PublishedAt) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	log.Printf("[INFO] total %d articles", len(all))
	return all, nil
}
