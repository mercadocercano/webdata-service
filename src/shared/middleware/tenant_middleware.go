package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/shared/database"
)

type contextKey string

const (
	TenantIDKey contextKey = "tenant_id"
)

func TenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Admin requests use global tenant
		if IsAdminFromContext(r.Context()) {
			ctx := context.WithValue(r.Context(), TenantIDKey, database.GlobalTenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		tenantIDStr := r.Header.Get("X-Tenant-ID")
		if tenantIDStr == "" {
			http.Error(w, `{"error":"X-Tenant-ID header is required"}`, http.StatusBadRequest)
			return
		}

		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			http.Error(w, `{"error":"X-Tenant-ID must be a valid UUID"}`, http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), TenantIDKey, tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TenantIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	tenantID, ok := ctx.Value(TenantIDKey).(uuid.UUID)
	return tenantID, ok
}
