-- 在 Supabase SQL Editor 中执行以下语句

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

-- 记录最近刷新时间，只存一行
CREATE TABLE IF NOT EXISTS refresh_state (
  id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  last_refresh TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO refresh_state (id, last_refresh) VALUES (1, NOW())
ON CONFLICT (id) DO NOTHING;
