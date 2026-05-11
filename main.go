package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"

	"ai-news-hub/hub"
)

//go:embed public
var publicFS embed.FS

//go:embed static
var staticFS embed.FS

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	http.HandleFunc("/api/", handleAPI)
	http.HandleFunc("/api", handleAPI)

	publicSub, _ := fs.Sub(publicFS, "public")
	http.Handle("/", http.FileServer(http.FS(publicSub)))

	staticSub, _ := fs.Sub(staticFS, "static")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	log.Println("starting on port " + port)
	http.ListenAndServe(":"+port, nil)
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
