package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	ClaimsKey    contextKey = "jwt_claims"
	IsAdminKey   contextKey = "is_admin"
)

type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Role     string    `json:"role"`
	jwt.RegisteredClaims
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check API Key auth (marketplace-admin)
		if authenticateByAPIKey(r) {
			ctx := context.WithValue(r.Context(), IsAdminKey, true)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// JWT auth
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"Authorization header is required"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			http.Error(w, `{"error":"Authorization header must be Bearer <token>"}`, http.StatusUnauthorized)
			return
		}

		tokenStr := parts[1]
		secret := os.Getenv("JWT_SECRET")

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, `{"error":"Invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func authenticateByAPIKey(r *http.Request) bool {
	apiKey := r.Header.Get("X-API-Key")
	userRole := r.Header.Get("X-User-Role")
	if apiKey == "" || userRole != "marketplace_admin" {
		return false
	}
	expectedKey := os.Getenv("MARKETPLACE_ADMIN_API_KEY")
	if expectedKey == "" {
		expectedKey = "marketplace-admin-key-2025"
	}
	return apiKey == expectedKey
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsKey).(*Claims)
	return claims, ok
}

func IsAdminFromContext(ctx context.Context) bool {
	isAdmin, _ := ctx.Value(IsAdminKey).(bool)
	return isAdmin
}
