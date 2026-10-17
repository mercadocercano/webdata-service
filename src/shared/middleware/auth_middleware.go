package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Context keys for identity left by AuthMiddleware. `RolesKey` holds the `[]string`
// of role slugs derived from the verified JWT — the SAME contract go-shared's
// VerifyPlatformJWT leaves in the Gin context (`roles`), so the chi RequireRoleHTTP
// helper below is the net/http mirror of go-shared's gin RequireRole.
const (
	ClaimsKey contextKey = "jwt_claims"
	RolesKey  contextKey = "roles"
)

// Claims carries the JWT identity claims webdata-service consumes.
//
// The authorization claim is `roles` ([]string of slugs) — the SAME claim iam-service
// emits today (auth/application/usecase/login.go and refresh_token.go populate
// `Roles []string`) and that go-shared VerifyPlatformJWT / catalog-service read. The
// legacy singular `role` string claim is gone: iam-service never emitted it, so it
// always deserialized to "" — the only "admin" path that worked was the API-key bypass
// removed in PLAT-E39 T9c1 (decisión D9b, condición O1 de @dev-security). Keeping a
// dead `Role string` would have silently broken the new system_admin derivation.
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Roles    []string  `json:"roles,omitempty"`
	jwt.RegisteredClaims
}

// AuthMiddleware verifies the JWT (HS256 legacy or RS256 by kid — ACC-E03 dual window) INSIDE the service and leaves the claims and
// the derived roles in the request context. It is the ONLY inbound identity path:
// the old `X-API-Key` + `X-User-Role: marketplace_admin` bypass with its baked
// `marketplace-admin-key-2025` fallback was removed in PLAT-E39 T9c1 (D9b) — a public
// secret by architecture (the PLAT-E30 §D2a disease). The role is derived EXCLUSIVELY
// from the `roles` claim of the already-verified JWT, never from a client/Kong header,
// which stays forgeable while PLAT-E37 (edge header stripping) does not exist (D3.1).
//
// Fail-closed by design (D3.2): without a valid token → 401. There is no RLS backstop
// on the platform tables reachable through admin routes, so the middleware is the
// single control — hence `jwt.WithExpirationRequired`: jwt/v5 only validates `exp` when
// the claim is present, so without this a token issued without `exp` would never expire,
// and a `system_admin` token without `exp` would be a permanent master key over the
// platform catalog (same fix as T1f in go-shared's VerifyPlatformJWT).
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// ACC-E03 T5b: regla dual (jwt_keyring.go) — HS256 → JWT_SECRET, RS256 → pública por kid
		// del keyring JWT_PUBLIC_KEY_DIR. Keyring roto → la rama RS256 queda sin claves (401).
		ring, ringErr := keyringFromEnv()
		if ringErr != nil {
			ring = nil
		}

		claims := &Claims{}
		token, err := jwt.NewParser(
			jwt.WithValidMethods(dualAcceptedMethods),
			jwt.WithExpirationRequired(), // T9c1/T1f: system_admin sin exp = llave maestra eterna (D3.2).
		).ParseWithClaims(tokenStr, claims, dualKeyFunc(secret, ring))
		if err != nil || !token.Valid {
			http.Error(w, `{"error":"Invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		ctx = context.WithValue(ctx, RolesKey, claims.Roles)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ClaimsFromContext returns the verified JWT claims, if present.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsKey).(*Claims)
	return claims, ok
}

// RolesFromContext returns the role slugs derived from the verified JWT. Returns nil
// when no verified token is in context (AuthMiddleware already rejected such requests
// with 401, so callers behind it can treat nil as "no roles").
func RolesFromContext(ctx context.Context) []string {
	roles, _ := ctx.Value(RolesKey).([]string)
	return roles
}

// HasRole reports whether the verified JWT carries the given role slug.
func HasRole(ctx context.Context, role string) bool {
	for _, r := range RolesFromContext(ctx) {
		if r == role {
			return true
		}
	}
	return false
}

// RequireRoleHTTP is the chi/net/http equivalent of go-shared's gin RequireRole: it
// admits a request only if the verified JWT carries one of `roles`. It MUST run after
// AuthMiddleware (which sets RolesKey); a request without a verified token never reaches
// it because AuthMiddleware returns 401 first.
//
// Deny-by-default (D3.2): a request whose token lacks every required role gets 403.
// This is the single authorization control for the platform/admin routes — there is no
// RLS backstop on the platform tables — so the gate is strict on purpose. Consolidating
// this helper into go-shared (net/http core + gin/chi adapters) is registered DRY debt
// from the T9c ADR, to be paid in a future go-shared release (same pattern as T1f→T1g);
// it is not the seventh ad-hoc mechanism: it is the same model (role from the verified
// JWT claim, in-service) as D3.1/D8/catalog, implemented for chi because webdata is chi.
func RequireRoleHTTP(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			granted := RolesFromContext(r.Context())
			for _, required := range roles {
				for _, g := range granted {
					if g == required {
						next.ServeHTTP(w, r)
						return
					}
				}
			}
			http.Error(w, `{"error":"forbidden: insufficient role"}`, http.StatusForbidden)
		})
	}
}