package hub

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	pool     *pgxpool.Pool
	poolErr  error
	poolOnce sync.Once
)

func getPool() (*pgxpool.Pool, error) {
	poolOnce.Do(func() {
		dsn := os.Getenv("DATABASE_URL")
		if dsn == "" {
			poolErr = fmt.Errorf("DATABASE_URL not set")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cfg, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			poolErr = fmt.Errorf("parse config: %w", err)
			return
		}
		cfg.ConnConfig.DefaultQueryExecMode = 4 // simple protocol for supabase pooler
		pool, poolErr = pgxpool.NewWithConfig(ctx, cfg)
		if poolErr != nil {
			return
		}
		if err := initDB(ctx); err != nil {
			pool.Close()
			pool = nil
			poolErr = fmt.Errorf("init db: %w", err)
		}
	})
	return pool, poolErr
}

func initDB(ctx context.Context) error {
	ddl := `
	CREATE TABLE IF NOT EXISTS articles (
	  id BIGSERIAL PRIMARY KEY,
	  title TEXT NOT NULL,
	  link TEXT NOT NULL,
	  summary TEXT DEFAULT '',
	  source TEXT NOT NULL,
	  published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	  image_url TEXT DEFAULT '',
	  tags TEXT[] DEFAULT '{}',
	  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_articles_source ON articles(source);
	CREATE INDEX IF NOT EXISTS idx_articles_published_at ON articles(published_at DESC);
	CREATE TABLE IF NOT EXISTS refresh_state (
	  id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
	  last_refresh TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	INSERT INTO refresh_state (id, last_refresh) VALUES (1, NOW())
	ON CONFLICT (id) DO NOTHING;
	`
	_, err := pool.Exec(ctx, ddl)
	return err
}

func GetAllArticles(source string) ([]Article, error) {
	p, err := getPool()
	if err != nil {
		return nil, err
	}
	var query string
	var args []any
	if source == "" {
		query = "SELECT title, link, summary, source, published_at, image_url, tags FROM articles ORDER BY published_at DESC"
	} else {
		query = "SELECT title, link, summary, source, published_at, image_url, tags FROM articles WHERE source = $1 ORDER BY published_at DESC"
		args = append(args, source)
	}
	rows, err := p.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query articles: %w", err)
	}
	defer rows.Close()
	var articles []Article
	for rows.Next() {
		var a Article
		if err := rows.Scan(&a.Title, &a.Link, &a.Summary, &a.Source, &a.PublishedAt, &a.ImageURL, &a.Tags); err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}
		articles = append(articles, a)
	}
	return articles, nil
}

func GetSources() ([]string, error) {
	p, err := getPool()
	if err != nil {
		return nil, err
	}
	rows, err := p.Query(context.Background(), "SELECT DISTINCT source FROM articles ORDER BY source")
	if err != nil {
		return nil, fmt.Errorf("query sources: %w", err)
	}
	defer rows.Close()
	var sources []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan source: %w", err)
		}
		sources = append(sources, s)
	}
	return sources, nil
}

func GetLastRefresh() (time.Time, error) {
	p, err := getPool()
	if err != nil {
		return time.Time{}, err
	}
	var t time.Time
	err = p.QueryRow(context.Background(), "SELECT last_refresh FROM refresh_state WHERE id = 1").Scan(&t)
	if err != nil {
		return time.Time{}, fmt.Errorf("query last_refresh: %w", err)
	}
	return t, nil
}

func RefreshAndSave() error {
	articles, err := RefreshAll()
	if err != nil {
		return err
	}
	return SaveArticles(articles)
}

func SaveArticles(articles []Article) error {
	p, err := getPool()
	if err != nil {
		return err
	}
	ctx := context.Background()
	tx, err := p.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, "DELETE FROM articles"); err != nil {
		return fmt.Errorf("delete articles: %w", err)
	}
	for _, a := range articles {
		if _, err := tx.Exec(ctx,
			`INSERT INTO articles (title, link, summary, source, published_at, image_url, tags)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			a.Title, a.Link, a.Summary, a.Source, a.PublishedAt, a.ImageURL, a.Tags,
		); err != nil {
			return fmt.Errorf("insert article: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, "UPDATE refresh_state SET last_refresh = NOW() WHERE id = 1"); err != nil {
		return fmt.Errorf("update last_refresh: %w", err)
	}
	return tx.Commit(ctx)
}
