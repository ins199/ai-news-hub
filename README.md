# AI Pulse — 科技资讯聚合

多源科技资讯实时聚合站，Go + Gin 驱动，前端金黑高奢风格。

## 快速启动

```bash
go build -o ai-pulse .
./ai-pulse
# 浏览器打开 http://localhost:8080
```

## 项目结构

```
.
├── main.go              # 入口，路由注册
├── models.go            # 数据结构 + 源注册
├── fetcher.go           # 数据获取（RSS + Algolia API）
├── static/
│   └── style.css        # 高奢金黑样式
├── templates/
│   └── index.html       # 前端页面
├── go.mod / go.sum      # Go 依赖
└── README.md
```

## 数据源

| 源 | 方式 | URL | 备注 |
|---|------|-----|------|
| IT之家 | RSS | `https://www.ithome.com/rss/` | 泛科技+数码，含图文详情 |
| 爱范儿 | RSS | `https://www.ifanr.com/feed` | 产品/数字潮牌视角 |
| 少数派 | RSS | `https://sspai.com/feed` | 效率工具/数字生活 |
| Hacker News | Algolia API | `search_by_date` | 国际科技/创业 |

添加新源：在 `models.go` 的 `sources` 切片中注册，在 `fetcher.go` 中实现对应的 `FetchFunc`。

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/` | 前端页面 |
| GET | `/api/news?source=xxx` | 文章列表（JSON），可选来源筛选 |
| GET | `/api/sources` | 当前数据源列表 |
| GET | `/api/refresh` | 手动触发全量刷新 |

## 刷新策略

- 启动时立即拉取一次
- 此后每 5 分钟自动刷新
- 可通过 `/api/refresh` 手动触发

## 依赖

- `github.com/gin-gonic/gin` — HTTP 框架
- `github.com/mmcdole/gofeed` — RSS 解析

## 注意事项

- IT之家、爱范儿、少数派 RSS 稳定性尚可但非官方承诺接口，如果失效需要更换 URL 或改用爬虫
- 少数派 RSS 不含配图，摘要较短
- Hacker News 走 Algolia 开放 API，无需 API Key
- 每个源限制 15-20 条，总计约 60 条，避免前端渲染压力过大
