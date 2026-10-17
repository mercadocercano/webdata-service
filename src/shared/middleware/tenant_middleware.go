package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const (
	TenantIDKey contextKey = "tenant_id"
)

func TenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Platform admin (system_admin) callers don't carry a tenant — by design their
		// data is no tenant's. The enrichment batch reads cross-tenant through the
		// explicit platform read path (T9c2, FindAllActiveByCategoryPlatform), not
		// through a sentinel tenant_id in context. So a system_admin request is
		// forwarded without a tenant_id and WITHOUT requiring X-Tenant-ID. The legacy
		// branch that set a sentinel tenant_id (the WHERE 1=1 bypass) was removed in
		// PLAT-E39 T9c1/T9c2 (D9b/D9a); there is no admin tenant in context anymore.
		if HasRole(r.Context(), "system_admin") {
			next.ServeHTTP(w, r)
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
