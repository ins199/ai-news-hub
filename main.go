package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"ai-news-hub/hub"
)

//go:embed public
var publicFS embed.FS

//go:embed static
var staticFS embed.FS

var indexTmpl = template.Must(template.New("index.html").ParseFS(publicFS, "public/index.html"))

var (
	robotsTxt  []byte
	sitemapXML []byte
)

func init() {
	var err error
	robotsTxt, err = publicFS.ReadFile("public/robots.txt")
	if err != nil {
		robotsTxt = []byte("User-agent: *\nAllow: /\n")
	}
	sitemapXML, err = publicFS.ReadFile("public/sitemap.xml")
	if err != nil {
		sitemapXML = []byte(`<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://ai-news-hub-xi.vercel.app</loc></url></urlset>`)
	}
}

type indexData struct {
	Articles        []hub.Article
	Sources         []string
	ArticlesJSON    template.JS
	SourcesJSON     template.JS
	LastRefreshJSON template.JS
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	http.HandleFunc("/api/", handleAPI)
	http.HandleFunc("/api", handleAPI)
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/robots.txt", handleRobots)
	http.HandleFunc("/sitemap.xml", handleSitemap)

	staticSub, _ := fs.Sub(staticFS, "static")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	log.Println("starting on port " + port)
	http.ListenAndServe(":"+port, nil)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	articles, err := hub.GetAllArticles("")
	if err != nil {
		log.Printf("[ERROR] index articles: %v", err)
		articles = []hub.Article{}
	}
	sources, err := hub.GetSources()
	if err != nil {
		log.Printf("[ERROR] index sources: %v", err)
		sources = []string{}
	}
	lastRefresh, _ := hub.GetLastRefresh()

	if articles == nil {
		articles = []hub.Article{}
	}
	if sources == nil {
		sources = []string{}
	}

	articlesJSON, _ := json.Marshal(articles)
	sourcesJSON, _ := json.Marshal(sources)
	lastRefreshJSON, _ := json.Marshal(lastRefresh.Format(time.RFC3339))

	data := indexData{
		Articles:        articles,
		Sources:         sources,
		ArticlesJSON:    template.JS(articlesJSON),
		SourcesJSON:     template.JS(sourcesJSON),
		LastRefreshJSON: template.JS(lastRefreshJSON),
	}

	var buf bytes.Buffer
	if err := indexTmpl.Execute(&buf, data); err != nil {
		log.Printf("[ERROR] template: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}

	// If no articles yet, tell JS to fetch via API
	if len(articles) == 0 {
		buf.Reset()
		indexTmpl.Execute(&buf, indexData{
			Sources:         sources,
			SourcesJSON:     template.JS(sourcesJSON),
			ArticlesJSON:    template.JS("null"),
			LastRefreshJSON: template.JS("null"),
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func handleRobots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write(robotsTxt)
}

func handleSitemap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	w.Write(sitemapXML)
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	switch action {
	case "news":
		handleNews(w, r)
	case "sources":
		handleSources(w)
	case "status":
		handleStatus(w)
	case "refresh":
		handleRefresh(w)
	default:
		http.NotFound(w, r)
	}
}

func handleNews(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	articles, err := hub.GetAllArticles(source)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sources, err := hub.GetSources()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if articles == nil {
		articles = []hub.Article{}
	}
	if sources == nil {
		sources = []string{}
	}
	lastRefresh, _ := hub.GetLastRefresh()
	json.NewEncoder(w).Encode(map[string]any{
		"articles":     articles,
		"sources":      sources,
		"last_refresh": lastRefresh,
	})
}

func handleSources(w http.ResponseWriter) {
	sources, err := hub.GetSources()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if sources == nil {
		sources = []string{}
	}
	json.NewEncoder(w).Encode(map[string]any{"sources": sources})
}

func handleStatus(w http.ResponseWriter) {
	lastRefresh, err := hub.GetLastRefresh()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	articles, err := hub.GetAllArticles("")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"last_refresh":  lastRefresh,
		"article_count": len(articles),
	})
}

func handleRefresh(w http.ResponseWriter) {
	articles, err := hub.RefreshAll()
	if err != nil {
		log.Printf("[ERROR] refresh: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := hub.SaveArticles(articles); err != nil {
		log.Printf("[ERROR] save: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
