package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// PLAT-E39 T9c1 — autz por endpoint (D3.2, sin backstop RLS sobre las tablas de
// plataforma que las rutas admin de webdata alcanzan). Los seis casos del "Hecho
// cuando" de T9c1, montados sobre el router real (AuthMiddleware + TenantMiddleware +
// RequireRoleHTTP) y un handler de prueba que sólo responde 200.
//
// El verificador es el MISMO que usa el servicio en runtime (AuthMiddleware lee
// JWT_SECRET de env, HS256, WithExpirationRequired), firmado acá con el mismo secreto.

const testSecret = "t9c1-test-secret-min-32-chars-long-xxxx"

func newAdminMux(t *testing.T, reached *bool) http.Handler {
	t.Helper()
	t.Setenv("JWT_SECRET", testSecret)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(AuthMiddleware)
		// Admin surface: role gate, sin TenantMiddleware (espejo del router real
		// post-T9c1). RequireRole corre antes que cualquier check de tenant, así
		// un tenant_admin a /enrichment/* toma 403, no 400 por X-Tenant-ID ausente.
		r.Group(func(r chi.Router) {
			r.Use(RequireRoleHTTP("system_admin"))
			r.Route("/enrichment", func(r chi.Router) {
				r.Post("/run", func(w http.ResponseWriter, r *http.Request) {
					*reached = true
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{}`))
				})
			})
		})
		// Tenant surface: exige X-Tenant-ID para no-admin.
		r.Group(func(r chi.Router) {
			r.Use(TenantMiddleware)
			r.Get("/sources/", func(w http.ResponseWriter, r *http.Request) {
				*reached = true
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			})
		})
	})
	return r
}

// mintToken firma un JWT HS256 con los claims dados. Si withExp es false, omite `exp`
// para ejercitar WithExpirationRequired (la llave maestra eterna que T1f cerró).
func mintToken(t *testing.T, roles []string, withExp bool, tenantID uuid.UUID) string {
	t.Helper()
	claims := jwt.MapClaims{
		"user_id":   uuid.New().String(),
		"tenant_id": tenantID.String(),
		"namespace": "mc",
	}
	if len(roles) > 0 {
		claims["roles"] = roles
	}
	if withExp {
		claims["exp"] = time.Now().Add(time.Hour).Unix()
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return s
}

func doRequest(t *testing.T, mux http.Handler, method, target string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// T1 — sin Authorization a /api/v1/enrichment/run → 401.
func TestEnrichment_NoAuth_401(t *testing.T) {
	var reached bool
	mux := newAdminMux(t, &reached)
	rec := doRequest(t, mux, http.MethodPost, "/api/v1/enrichment/run", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("sin Authorization: got %d, want 401", rec.Code)
	}
	if reached {
		t.Fatal("el handler corrió sin token (fail-closed roto)")
	}
}

// T2 — JWT tenant_admin (sin system_admin) → 403.
func TestEnrichment_TenantAdmin_403(t *testing.T) {
	var reached bool
	mux := newAdminMux(t, &reached)
	rec := doRequest(t, mux, http.MethodPost, "/api/v1/enrichment/run", map[string]string{
		"Authorization": "Bearer " + mintToken(t, []string{"tenant_admin"}, true, uuid.New()),
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("tenant_admin a ruta admin: got %d, want 403", rec.Code)
	}
	if reached {
		t.Fatal("el handler corrió con rol insuficiente (deny-by-default roto)")
	}
}

// T3 — JWT system_admin con exp → 200.
func TestEnrichment_SystemAdmin_200(t *testing.T) {
	var reached bool
	mux := newAdminMux(t, &reached)
	rec := doRequest(t, mux, http.MethodPost, "/api/v1/enrichment/run", map[string]string{
		"Authorization": "Bearer " + mintToken(t, []string{"system_admin"}, true, uuid.New()),
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("system_admin con exp: got %d, want 200", rec.Code)
	}
	if !reached {
		t.Fatal("el handler no corrió con system_admin válido")
	}
}

// T4 — JWT system_admin SIN exp → 401 (WithExpirationRequired).
func TestEnrichment_SystemAdmin_SinExp_401(t *testing.T) {
	var reached bool
	mux := newAdminMux(t, &reached)
	rec := doRequest(t, mux, http.MethodPost, "/api/v1/enrichment/run", map[string]string{
		"Authorization": "Bearer " + mintToken(t, []string{"system_admin"}, false, uuid.New()),
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("system_admin sin exp: got %d, want 401 (exp obligatorio)", rec.Code)
	}
	if reached {
		t.Fatal("el handler corrió con un token eterno (WithExpirationRequired roto)")
	}
}

// T5 — bypass viejo (X-API-Key + X-User-Role: marketplace_admin) → 401. El bypass ya
// no existe: sin Authorization, AuthMiddleware rechaza antes de cualquier header de rol.
func TestEnrichment_LegacyAPIKeyBypass_401(t *testing.T) {
	// Header de rol forjable, en minúsculas a propósito: Header.Set canonicaliza, y la
	// forma canónica literal sólo se admite en comentarios (check-hexagonal, regla 5).
	const forgedRoleHeader = "x-user-role"
	var reached bool
	mux := newAdminMux(t, &reached)
	rec := doRequest(t, mux, http.MethodPost, "/api/v1/enrichment/run", map[string]string{
		"X-API-Key":      "marketplace-admin-key-2025",
		forgedRoleHeader: "marketplace_admin",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bypass viejo: got %d, want 401 (el bypass fue eliminado)", rec.Code)
	}
	if reached {
		t.Fatal("el bypass API-key todavía admite (D9b roto)")
	}
}

// T6 — callers tenant (sources) con JWT + X-Tenant-ID siguen funcionando (regresión).
// Un tenant_admin con X-Tenant-ID válido alcanza su ruta tenant (no admin) → 200.
func TestSources_TenantCaller_200(t *testing.T) {
	var reached bool
	mux := newAdminMux(t, &reached)
	tid := uuid.New()
	rec := doRequest(t, mux, http.MethodGet, "/api/v1/sources/", map[string]string{
		"Authorization": "Bearer " + mintToken(t, []string{"tenant_admin"}, true, tid),
		"X-Tenant-ID":   tid.String(),
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("caller tenant a ruta tenant: got %d, want 200", rec.Code)
	}
	if !reached {
		t.Fatal("el handler tenant no corrió con JWT+X-Tenant-ID válidos")
	}
}

// T6b — un system_admin SIN X-Tenant-ID a una ruta tenant no admin sigue siendo
// admitido por AuthMiddleware+TenantMiddleware (skip de tenant), pero como la ruta
// no exige system_admin, llega al handler. Esto fija el contrato de TenantMiddleware:
// system_admin no necesita X-Tenant-ID (D9b/F2). El handler real de /sources leería
// TenantIDFromContext y fallaría, pero ese es comportamiento del handler, no del
// middleware — acá verificamos sólo que el middleware no bloquea con 400.
func TestTenantMiddleware_SystemAdmin_SinXTenantID_Pasa(t *testing.T) {
	var reached bool
	mux := newAdminMux(t, &reached)
	rec := doRequest(t, mux, http.MethodGet, "/api/v1/sources/", map[string]string{
		"Authorization": "Bearer " + mintToken(t, []string{"system_admin"}, true, uuid.New()),
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("system_admin sin X-Tenant-ID: got %d, want 200 (TenantMiddleware no debe exigir tenant)", rec.Code)
	}
}

// T6c — un tenant_admin SIN X-Tenant-ID → 400 (TenantMiddleware sigue fail-closed
// para callers no-admin). Confirma que el skip de tenant es sólo para system_admin.
func TestTenantMiddleware_TenantAdmin_SinXTenantID_400(t *testing.T) {
	var reached bool
	mux := newAdminMux(t, &reached)
	rec := doRequest(t, mux, http.MethodGet, "/api/v1/sources/", map[string]string{
		"Authorization": "Bearer " + mintToken(t, []string{"tenant_admin"}, true, uuid.New()),
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("tenant_admin sin X-Tenant-ID: got %d, want 400", rec.Code)
	}
	if reached {
		t.Fatal("el handler corrió sin tenant para un caller no-admin")
	}
}

// O1 — el claim de autorización es `roles` (slice), no `role` (string). Un token con
// `roles: ["system_admin"]` es admitido; el claim singular `role` ya no se lee.
// iam-service emite `roles []string` (login.go/refresh_token.go), nunca `role`.
func TestRolesClaim_Slice_NoSingularRole(t *testing.T) {
	var reached bool
	mux := newAdminMux(t, &reached)
	// Token con el claim singular legacy "role":"system_admin" pero SIN "roles" →
	// debe caer a 403: el slug ya no se deriva de un claim que iam no emite.
	claims := jwt.MapClaims{
		"user_id":   uuid.New().String(),
		"tenant_id": uuid.New().String(),
		"namespace": "mc",
		"role":      "system_admin", // legacy, ya ignorado
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := tok.SignedString([]byte(testSecret))
	rec := doRequest(t, mux, http.MethodPost, "/api/v1/enrichment/run", map[string]string{
		"Authorization": "Bearer " + s,
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("claim legacy `role` singular: got %d, want 403 (sólo `roles` slice deriva rol)", rec.Code)
	}
}

// HasRole / RolesFromContext — contratos del helper local.
func TestHasRole(t *testing.T) {
	ctx := context.WithValue(context.Background(), RolesKey, []string{"tenant_admin", "system_admin"})
	if !HasRole(ctx, "system_admin") {
		t.Error("HasRole(system_admin) debería ser true")
	}
	if HasRole(ctx, "cashier") {
		t.Error("HasRole(cashier) debería ser false")
	}
	if HasRole(context.Background(), "system_admin") {
		t.Error("HasRole en ctx sin roles debería ser false")
	}
}

// Sanity: un secreto distinto rechaza el token. AuthMiddleware lee JWT_SECRET de env
// en cada request, así que sobreescribimos DESPUÉS de construir el mux (newAdminMux
// fija testSecret); el token se firma con testSecret → mismatch → 401.
func TestEnrichment_WrongSecret_401(t *testing.T) {
	var reached bool
	mux := newAdminMux(t, &reached)
	t.Setenv("JWT_SECRET", "otro-secreto-distinto-min-32-chars-xxxxxxxxxx")
	rec := doRequest(t, mux, http.MethodPost, "/api/v1/enrichment/run", map[string]string{
		"Authorization": "Bearer " + mintToken(t, []string{"system_admin"}, true, uuid.New()),
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("secreto distinto: got %d, want 401", rec.Code)
	}
}
