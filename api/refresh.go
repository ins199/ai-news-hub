package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"ai-news-hub/hub"
)

func Handler(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
