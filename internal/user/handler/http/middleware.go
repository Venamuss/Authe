package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"authe/internal/user"
)

type contextKey string

const (
	usernameContextKey contextKey = "username"
)

type Cache interface {
	CheckBlacklistToken(ctx context.Context, token string) error
	CheckTimeout(ctx context.Context, ip string) (int, error)
}

type TokenManager interface {
	CreateToken(username string) (string, error)
	VerifyToken(tokenString string) error
	ExtractClaimsWithMap(tokenString string) (jwt.MapClaims, error)
}

type Middleware struct {
	cache        Cache
	tokenManager TokenManager
	rateLimitMax int
}

func NewMiddleware(cache Cache, tokenManager TokenManager, rateLimitMax int) *Middleware {
	return &Middleware{
		cache:        cache,
		tokenManager: tokenManager,
		rateLimitMax: rateLimitMax,
	}
}

func (middleware *Middleware) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "no token provided")
			return
		}
		token = strings.TrimPrefix(token, "Bearer ")

		if err := middleware.cache.CheckBlacklistToken(r.Context(), token); errors.Is(err, user.TokenBlacklisted) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "token is blacklisted")
			return
		} else if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "failed to check token: ", err)
			return
		}

		claims, err := middleware.tokenManager.ExtractClaimsWithMap(token)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "invalid token")
			return
		}
		username, ok := claims["username"].(string)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), usernameContextKey, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func (middleware *Middleware) Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start))
	})
}

func (middleware *Middleware) Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"err", err,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error": "internal"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (middleware *Middleware) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		n, err := middleware.cache.CheckTimeout(r.Context(), ip)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "failed to check timeout: ", err)
			return
		}
		if n > middleware.rateLimitMax {
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, "too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	if xRealIP := r.Header.Get("X-Real-IP"); xRealIP != "" {
		return xRealIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
