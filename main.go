package main

import (
	"log"
	"net/http"
	"time"

	"ai-news-hub/hub"

	"github.com/gin-gonic/gin"
)

func main() {
	go startRefresher(5 * time.Minute)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.StaticFile("/", "./public/index.html")
	r.Static("/static", "./static")

	r.GET("/api/news", func(c *gin.Context) {
		source := c.Query("source")
		articles, err := hub.GetAllArticles(source)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		sources, err := hub.GetSources()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if articles == nil {
			articles = []hub.Article{}
		}
		if sources == nil {
			sources = []string{}
		}
		c.JSON(http.StatusOK, gin.H{
			"articles": articles,
			"sources":  sources,
		})
	})

	r.GET("/api/sources", func(c *gin.Context) {
		sources, err := hub.GetSources()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"sources": sources})
	})

	r.GET("/api/status", func(c *gin.Context) {
		lastRefresh, err := hub.GetLastRefresh()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		articles, err := hub.GetAllArticles("")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"last_refresh":  lastRefresh,
			"article_count": len(articles),
		})
	})

	r.GET("/api/refresh", func(c *gin.Context) {
		articles, err := hub.RefreshAll()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := hub.SaveArticles(articles); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	log.Println("starting server on :8080")
	r.Run(":8080")
}

func startRefresher(interval time.Duration) {
	hub.RefreshAndSave()
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			hub.RefreshAndSave()
		}
	}()
}
