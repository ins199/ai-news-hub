package handler

import (
	"encoding/json"
	"net/http"

	"ai-news-hub/hub"
)

func Handler(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"last_refresh": lastRefresh,
		"article_count": len(articles),
	})
}
