// Package testsupport es el fixture RS256 de los tests de webdata-service. Desde ACC-E03 T6 la
// flota sólo acepta RS256 por kid (go-shared sin rama HMAC), así que los tests que antes firmaban
// HS256 con un secreto local firman con esta clave y apuntan JWT_PUBLIC_KEY_DIR a su keyring.
// Sólo lo importan archivos _test.go.
package testsupport

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"sync"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hornosg/go-shared/infrastructure/jwtkeyring"
)

var (
	once sync.Once
	key  *rsa.PrivateKey
	dir  string
)

func fixture() (*rsa.PrivateKey, string) {
	once.Do(func() {
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
		der, err := x509.MarshalPKIXPublicKey(&k.PublicKey)
		if err != nil {
			panic(err)
		}
		d, err := os.MkdirTemp("", "webdata-keyring-*")
		if err != nil {
			panic(err)
		}
		pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
		if err := os.WriteFile(filepath.Join(d, jwtkeyring.KID(&k.PublicKey)+".pem"), pemBytes, 0o600); err != nil {
			panic(err)
		}
		key, dir = k, d
	})
	return key, dir
}

// KeyringDir es el JWT_PUBLIC_KEY_DIR con la pública del fixture.
func KeyringDir() string {
	_, d := fixture()
	return d
}

// PrivateKey firma los tokens del fixture.
func PrivateKey() *rsa.PrivateKey {
	k, _ := fixture()
	return k
}

// NewRS256Token arma un token RS256 con el kid de la pública del keyring.
func NewRS256Token(claims jwt.Claims) *jwt.Token {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = jwtkeyring.KID(&PrivateKey().PublicKey)
	return token
}
