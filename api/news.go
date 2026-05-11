package handler

import (
	"encoding/json"
	"net/http"

	"ai-news-hub/hub"
)

func Handler(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"articles": articles,
		"sources":  sources,
	})
}
