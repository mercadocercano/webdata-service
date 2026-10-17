package middleware

import (
	"os"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v5"

	"github.com/hornosg/go-shared/infrastructure/jwtkeyring"
)

// ACC-E03 T5b: AuthMiddleware es un verificador PROPIO de webdata (chi, no gin). La regla dual
// (HS256 → JWT_SECRET, RS256 → pública por kid de JWT_PUBLIC_KEY_DIR) vive en
// go-shared/infrastructure/jwtkeyring (v0.18.0); acá sólo se resuelve el keyring desde el entorno.
// T6 retira la rama HMAC.

// PublicKeyring es el keyring compartido de go-shared.
type PublicKeyring = jwtkeyring.Keyring

var dualAcceptedMethods = jwtkeyring.AcceptedMethods

// keyringFromEnv resuelve JWT_PUBLIC_KEY_DIR con la validación de arranque fail-closed
// (ADR-003 addendum §C). Dir configurado → tiene que cargar; dir vacío con GIN_MODE=release →
// error; dir vacío fuera de release → (nil, nil): la rama RS256 rechaza todo.
//
// Se cachea por directorio: AuthMiddleware lee el env por request (igual que JWT_SECRET) y
// recargar los PEM en cada request sería I/O gratuito.
func keyringFromEnv() (*PublicKeyring, error) {
	dir := strings.TrimSpace(os.Getenv("JWT_PUBLIC_KEY_DIR"))
	if dir == "" {
		return jwtkeyring.Resolve("", os.Getenv("GIN_MODE") == "release")
	}
	keyringMu.Lock()
	defer keyringMu.Unlock()
	if ring, ok := keyringByDir[dir]; ok {
		return ring, nil
	}
	ring, err := jwtkeyring.Load(dir)
	if err != nil {
		return nil, err
	}
	keyringByDir[dir] = ring
	return ring, nil
}

var (
	keyringMu    sync.Mutex
	keyringByDir = map[string]*PublicKeyring{}
)

// ValidateKeyringFromEnv es el fail-fast de arranque para cmd/api/main.go.
func ValidateKeyringFromEnv() error {
	_, err := keyringFromEnv()
	return err
}

func dualKeyFunc(secret string, ring *PublicKeyring) jwt.Keyfunc {
	return jwtkeyring.Keyfunc(secret, ring)
}

func parseDual(tokenStr string, claims jwt.Claims, secret string, ring *PublicKeyring, opts ...jwt.ParserOption) error {
	return jwtkeyring.ParseDual(tokenStr, claims, secret, ring, opts...)
}

// LoadPublicKeyring lee todos los `*.pem` de dir (ver jwtkeyring.Load).
func LoadPublicKeyring(dir string) (*PublicKeyring, error) { return jwtkeyring.Load(dir) }

// KeyringKID deriva el kid igual que iam-service (ver jwtkeyring.KID).
var KeyringKID = jwtkeyring.KID
