package http

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (string, error)
}

type Router struct {
	validator      TokenValidator
	authURL        *url.URL
	agentsURL      *url.URL
	chatsURL       *url.URL
	allowedOrigins []string
}

func NewRouter(v TokenValidator, authAddr, agentsAddr, chatsAddr, allowedOriginsStr string) (*Router, error) {
	parsedAuth, err := url.Parse(authAddr)
	if err != nil {
		return nil, err
	}
	parsedAgents, err := url.Parse(agentsAddr)
	if err != nil {
		return nil, err
	}
	parsedChats, err := url.Parse(chatsAddr)
	if err != nil {
		return nil, err
	}

	// Разрезаем строку из конфига по запятым на чистый массив доменов
	origins := strings.Split(allowedOriginsStr, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	return &Router{
		validator:      v,
		authURL:        parsedAuth,
		agentsURL:      parsedAgents,
		chatsURL:       parsedChats,
		allowedOrigins: origins,
	}, nil
}

func (rt *Router) RegisterRoutes() *chi.Mux {
	r := chi.NewRouter()

	// 1. НАСТРОЙКА CORS MIDDLEWARE (Должна идти САМОЙ ПЕРВОЙ в цепочке!)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   rt.allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Request-Id"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	authProxy := httputil.NewSingleHostReverseProxy(rt.authURL)

	createSecureProxy := func(targetURL *url.URL) *httputil.ReverseProxy {
		return &httputil.ReverseProxy{
			FlushInterval: 100 * time.Millisecond, // Стриминг для SSE
			Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(targetURL)
				userID := GetUserID(pr.In.Context())
				if userID != "" {
					pr.Out.Header.Set("X-User-Id", userID)
				}
			},
		}
	}

	agentsProxy := createSecureProxy(rt.agentsURL)
	chatsProxy := createSecureProxy(rt.chatsURL)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Публичные маршруты
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", authProxy.ServeHTTP)
		r.Post("/login", authProxy.ServeHTTP)
		r.Post("/anonimous", authProxy.ServeHTTP)
	})

	// Защищенные маршруты
	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware(rt.validator))

		r.Route("/api/v1/agents", func(r chi.Router) {
			r.HandleFunc("/*", agentsProxy.ServeHTTP)
		})

		r.Route("/api/v1/chats", func(r chi.Router) {
			r.Post("/sessions", chatsProxy.ServeHTTP)              // Создать новый трэд
			r.Get("/sessions", chatsProxy.ServeHTTP)               // Получить список чатов для Sidebar
			r.Put("/sessions/{id}/title", chatsProxy.ServeHTTP)    // Переименовать чат
			r.Delete("/sessions/{id}", chatsProxy.ServeHTTP)       // Удалить чат
			r.Get("/sessions/{id}/messages", chatsProxy.ServeHTTP) // Выгрузить историю для
		})
	})

	return r
}
