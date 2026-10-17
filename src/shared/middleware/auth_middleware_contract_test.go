package middleware

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

// ACC-E03 T5b — tests de CONTRATO (fase TEST, @dev-qa) sobre AuthMiddleware (verificador propio,
// control único de admin de webdata). Completan lo que los tests de EXEC no ejercitan.

func wdContractSign(t *testing.T, m jwt.SigningMethod, key interface{}, kid string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(m, claims)
	if kid != "" {
		tok.Header["kid"] = kid
	}
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign %s: %v", m.Alg(), err)
	}
	return s
}

func wdContractMux(t *testing.T, secret, dir string) http.Handler {
	t.Helper()
	t.Setenv("JWT_SECRET", secret)
	t.Setenv("JWT_PUBLIC_KEY_DIR", dir)
	r := chi.NewRouter()
	r.Use(AuthMiddleware)
	r.Get("/probe", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	return r
}

func TestContratoT5b_AuthMiddleware_OtrosAlgsConLaClaveCorrecta_401(t *testing.T) {
	key := dualKey(t)
	h := dualMux(t, &key.PublicKey)
	kid := KeyringKID(&key.PublicKey)
	for name, tok := range map[string]string{
		"RS384": wdContractSign(t, jwt.SigningMethodRS384, key, kid, dualClaims()),
		"PS256": wdContractSign(t, jwt.SigningMethodPS256, key, kid, dualClaims()),
		"HS512": wdContractSign(t, jwt.SigningMethodHS512, []byte(testSecret), "", dualClaims()),
	} {
		if w := dualDo(h, tok); w.Code != http.StatusUnauthorized {
			t.Errorf("%s: code=%d, se espera 401", name, w.Code)
		}
	}
}

func TestContratoT5b_AuthMiddleware_RS256Expirado_401(t *testing.T) {
	key := dualKey(t)
	h := dualMux(t, &key.PublicKey)
	c := dualClaims()
	c["exp"] = time.Now().Add(-time.Minute).Unix()
	if w := dualDo(h, wdContractSign(t, jwt.SigningMethodRS256, key, KeyringKID(&key.PublicKey), c)); w.Code != http.StatusUnauthorized {
		t.Fatalf("RS256 expirado: code=%d, se espera 401", w.Code)
	}
}

// Control único de admin: un RS256 válido SIN system_admin no puede llegar al handler como admin
// (dualMux responde 418 si el rol no quedó en contexto; 200 sería escalada).
func TestContratoT5b_AuthMiddleware_RS256SinRolAdmin_NoEsAdmin(t *testing.T) {
	key := dualKey(t)
	h := dualMux(t, &key.PublicKey)
	c := dualClaims()
	c["roles"] = []string{"tenant_admin"}
	if w := dualDo(h, wdContractSign(t, jwt.SigningMethodRS256, key, KeyringKID(&key.PublicKey), c)); w.Code == http.StatusOK {
		t.Fatalf("RS256 sin system_admin quedó como admin: code=%d", w.Code)
	}
}

func TestContratoT5b_AuthMiddleware_KidDelMaterialNoDelNombre(t *testing.T) {
	key := dualKey(t)
	dir := t.TempDir()
	const nombre = "acc-000000000000"
	if err := os.WriteFile(filepath.Join(dir, nombre+".pem"), dualPubPEM(t, &key.PublicKey), 0o600); err != nil {
		t.Fatalf("write pem: %v", err)
	}
	h := wdContractMux(t, testSecret, dir)
	if w := dualDo(h, wdContractSign(t, jwt.SigningMethodRS256, key, nombre, dualClaims())); w.Code != http.StatusUnauthorized {
		t.Errorf("kid = nombre del archivo: code=%d, se espera 401", w.Code)
	}
	if w := dualDo(h, wdContractSign(t, jwt.SigningMethodRS256, key, KeyringKID(&key.PublicKey), dualClaims())); w.Code != http.StatusOK {
		t.Errorf("kid derivado: code=%d, se espera 200", w.Code)
	}
}

// Jamás HMAC con clave vacía: JWT_SECRET vacío → un HS256 firmado con "" no autentica.
func TestContratoT5b_AuthMiddleware_SecretoVacio_HS256Rechazado(t *testing.T) {
	key := dualKey(t)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "k.pem"), dualPubPEM(t, &key.PublicKey), 0o600)
	h := wdContractMux(t, "", dir)
	if w := dualDo(h, wdContractSign(t, jwt.SigningMethodHS256, []byte(""), "", dualClaims())); w.Code != http.StatusUnauthorized {
		t.Fatalf("HS256 con secreto vacío: code=%d, se espera 401", w.Code)
	}
}

// Keyring roto en runtime (el fail-fast de arranque está en main.go): la rama RS256 tiene que
// fallar cerrada — ni siquiera un token con la clave "buena" del archivo roto pasa.
func TestContratoT5b_AuthMiddleware_KeyringRoto_RS256Rechazado(t *testing.T) {
	key := dualKey(t)
	pemBytes := dualPubPEM(t, &key.PublicKey)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.pem"), pemBytes, 0o600)
	_ = os.WriteFile(filepath.Join(dir, "b.pem"), pemBytes, 0o600) // kid duplicado
	h := wdContractMux(t, testSecret, dir)
	if w := dualDo(h, wdContractSign(t, jwt.SigningMethodRS256, key, KeyringKID(&key.PublicKey), dualClaims())); w.Code != http.StatusUnauthorized {
		t.Fatalf("keyring con kid duplicado: code=%d, se espera 401", w.Code)
	}
}
