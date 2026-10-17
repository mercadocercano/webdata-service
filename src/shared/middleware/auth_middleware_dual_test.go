package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ACC-E03 T5b — AuthMiddleware acepta RS256 por kid (keyring JWT_PUBLIC_KEY_DIR) además del
// HS256 legacy, y rechaza otra clave, alg:none y la confusión RS/HS con 401.

func dualKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa: %v", err)
	}
	return k
}

func dualPubPEM(t *testing.T, pub *rsa.PublicKey) []byte {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal pub: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
}

// dualMux monta AuthMiddleware con el keyring de pubs y devuelve el user_id que dejó en contexto.
func dualMux(t *testing.T, pubs ...*rsa.PublicKey) http.Handler {
	t.Helper()
	dir := t.TempDir()
	for _, pub := range pubs {
		if err := os.WriteFile(filepath.Join(dir, KeyringKID(pub)+".pem"), dualPubPEM(t, pub), 0o600); err != nil {
			t.Fatalf("write pem: %v", err)
		}
	}
	t.Setenv("JWT_SECRET", testSecret)
	t.Setenv("JWT_PUBLIC_KEY_DIR", dir)
	r := chi.NewRouter()
	r.Use(AuthMiddleware)
	r.Get("/probe", func(w http.ResponseWriter, r *http.Request) {
		c, ok := ClaimsFromContext(r.Context())
		if !ok || !HasRole(r.Context(), "system_admin") {
			w.WriteHeader(http.StatusTeapot)
			return
		}
		_, _ = w.Write([]byte(c.UserID.String()))
	})
	return r
}

var dualUserID = uuid.MustParse("11111111-2222-3333-4444-555555555555")

func dualClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"user_id":   dualUserID.String(),
		"tenant_id": uuid.NewString(),
		"roles":     []string{"system_admin"},
		"iss":       "iam-service",
		"exp":       time.Now().Add(15 * time.Minute).Unix(),
	}
}

func dualSign(t *testing.T, m jwt.SigningMethod, key interface{}, kid string) string {
	t.Helper()
	tok := jwt.NewWithClaims(m, dualClaims())
	if kid != "" {
		tok.Header["kid"] = kid
	}
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign %s: %v", m.Alg(), err)
	}
	return s
}

func dualDo(h http.Handler, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestAuthMiddlewareDual_RS256Valido_PasaYPueblaClaims(t *testing.T) {
	key := dualKey(t)
	h := dualMux(t, &key.PublicKey)
	w := dualDo(h, dualSign(t, jwt.SigningMethodRS256, key, KeyringKID(&key.PublicKey)))
	if w.Code != http.StatusOK || w.Body.String() != dualUserID.String() {
		t.Fatalf("RS256 válido: code=%d body=%q", w.Code, w.Body.String())
	}
}

func TestAuthMiddlewareDual_HS256Legacy_SigueAceptado(t *testing.T) {
	key := dualKey(t)
	h := dualMux(t, &key.PublicKey)
	if w := dualDo(h, dualSign(t, jwt.SigningMethodHS256, []byte(testSecret), "")); w.Code != http.StatusOK {
		t.Fatalf("HS256 legacy durante la ventana dual: code=%d", w.Code)
	}
}

func TestAuthMiddlewareDual_Negativos_401(t *testing.T) {
	key, other := dualKey(t), dualKey(t)
	h := dualMux(t, &key.PublicKey)
	kid := KeyringKID(&key.PublicKey)
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, dualClaims()).SignedString(jwt.UnsafeAllowNoneSignatureType)

	for name, tok := range map[string]string{
		"RS256 firmado con OTRA clave":              dualSign(t, jwt.SigningMethodRS256, other, kid),
		"RS256 con kid desconocido":                 dualSign(t, jwt.SigningMethodRS256, other, KeyringKID(&other.PublicKey)),
		"alg:none":                                  none,
		"HS256 con la pública como secreto (RS/HS)": dualSign(t, jwt.SigningMethodHS256, dualPubPEM(t, &key.PublicKey), kid),
	} {
		if w := dualDo(h, tok); w.Code != http.StatusUnauthorized {
			t.Errorf("%s: code=%d, se espera 401", name, w.Code)
		}
	}
}

func TestAuthMiddlewareDual_ArranqueFailClosed(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY_DIR", "")
	t.Setenv("GIN_MODE", "release")
	if err := ValidateKeyringFromEnv(); err == nil {
		t.Error("release sin keyring: se esperaba error de arranque")
	}
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("JWT_PUBLIC_KEY_DIR", t.TempDir())
	if err := ValidateKeyringFromEnv(); err == nil {
		t.Error("keyring vacío: se esperaba error de arranque")
	}
}
