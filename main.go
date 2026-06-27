package main

import (
	"crypto/subtle"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/wtz44/mimo-gateway/internal/config"
	"github.com/wtz44/mimo-gateway/internal/convstore"
	"github.com/wtz44/mimo-gateway/internal/handler"
	"github.com/wtz44/mimo-gateway/internal/pool"
	"github.com/wtz44/mimo-gateway/internal/stats"
)

//go:embed static/*
var staticFiles embed.FS

// 并发上限：自用场景默认 8，可通过环境变量 MAX_CONCURRENCY 调整
const defaultMaxConcurrency = 8

func main() {
	configPath := "config.json"
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		configPath = p
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 弱口令警告
	if cfg.APIKey == "sk-mimo" {
		log.Printf("⚠️  警告: API Key 仍为默认值 \"sk-mimo\"，请在管理面板或 config.json 中修改为强随机值")
	}

	// 初始化统计追踪
	stats.Init("data/stats.json")
	stats.InitConvStore("data/conversations.json")

	// 初始化账号池
	accountPool := pool.New(cfg.Accounts)

	// 路由
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	// 自用场景：CORS 不再放行凭证（与通配 origin 共存会被浏览器拒绝，且无必要）
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// 全局并发上限中间件
	maxConc := defaultMaxConcurrency
	if v := os.Getenv("MAX_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxConc = n
		}
	}
	concLimiter := newConcurrencyLimiter(maxConc)

	convStore := convstore.New()
	chatHandler := handler.NewChatHandler(accountPool, convStore)
	messagesHandler := handler.NewMessagesHandler(accountPool, convStore)
	adminHandler := handler.NewAdminHandler(accountPool)

	// OpenAI 兼容
	r.Route("/v1", func(r chi.Router) {
		r.Use(apiKeyAuth)
		r.Use(concLimiter.handler)
		r.Post("/chat/completions", chatHandler.Handle)
		r.Get("/models", handler.ModelsHandler)
		r.Get("/models/{id}", handler.ModelsHandler)
		r.Post("/messages", messagesHandler.Handle)
		r.Post("/messages/count_tokens", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"input_tokens":0}`))
		})
	})

	// Anthropic 快捷路径
	r.With(apiKeyAuth, concLimiter.handler).Post("/anthropic/v1/messages", messagesHandler.Handle)

	// 管理接口（同样需要鉴权，复用 API Key）
	r.Route("/admin/api", func(r chi.Router) {
		r.Use(apiKeyAuth)
		r.Get("/config", adminHandler.GetConfig)
		r.Post("/config", adminHandler.UpdateConfig)
		r.Post("/accounts", adminHandler.AddAccount)
		r.Delete("/accounts", adminHandler.DeleteAccount)
		r.Get("/health", adminHandler.HealthCheck)
		r.Get("/stats", adminHandler.GetStats)
	})

	// 前端 SPA
	staticFS, _ := fs.Sub(staticFiles, "static")
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		filePath := path[1:]
		if data, err := fs.ReadFile(staticFS, filePath); err == nil {
			ext := filePath[len(filePath)-len(getExt(filePath)):]
			w.Header().Set("Content-Type", contentType(ext))
			w.Write(data)
			return
		}
		if data, err := fs.ReadFile(staticFS, "index.html"); err == nil {
			w.Header().Set("Content-Type", "text/html")
			w.Write(data)
			return
		}
		http.NotFound(w, r)
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("🚀 MiMo Gateway starting on http://localhost%s", addr)
	log.Printf("   Accounts: %d", accountPool.Count())

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// extractAPIKey 从常见请求头中提取 API Key
func extractAPIKey(r *http.Request) string {
	key := r.Header.Get("Authorization")
	if key == "" {
		key = r.Header.Get("api-key")
	}
	if key == "" {
		// Anthropic 官方客户端使用 x-api-key
		key = r.Header.Get("x-api-key")
	}
	if len(key) > 7 && key[:7] == "Bearer " {
		key = key[7:]
	}
	return key
}

func apiKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := extractAPIKey(r)
		currentCfg := config.Get()
		// constant-time 比较避免时序攻击
		if subtle.ConstantTimeCompare([]byte(key), []byte(currentCfg.APIKey)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"Invalid API key","type":"authentication_error"}}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// concurrencyLimiter 简单的计数信号量，限制全局并发请求数
type concurrencyLimiter struct {
	sem chan struct{}
}

func newConcurrencyLimiter(max int) *concurrencyLimiter {
	return &concurrencyLimiter{sem: make(chan struct{}, max)}
}

func (l *concurrencyLimiter) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case l.sem <- struct{}{}:
			defer func() { <-l.sem }()
			next.ServeHTTP(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":{"message":"server busy, try again later","type":"rate_limit_error"}}`))
		}
	})
}

func getExt(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i:]
		}
		if path[i] == '/' {
			return ""
		}
	}
	return ""
}

func contentType(ext string) string {
	switch ext {
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}
