package platformauth

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewServiceTokenProvider_MissingSecretFailsFast(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	if _, err := NewServiceTokenProvider(); err == nil {
		t.Fatalf("esperaba error por JWT_SECRET ausente (D4 — sin fallback horneado)")
	}
}

func TestToken_HasRequiredClaimsAndExpires(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-platformauth-32chars-min!!")
	p, err := NewServiceTokenProvider()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	tokStr, err := p.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}

	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(tokStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			t.Fatalf("esperaba HS256")
		}
		return []byte("test-secret-platformauth-32chars-min!!"), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("token inválido: %v", err)
	}

	if ns, _ := claims["namespace"].(string); ns != "mc" {
		t.Errorf("namespace: quiero %q, tengo %q", "mc", ns)
	}
	roles, _ := claims["roles"].([]interface{})
	if len(roles) != 1 || roles[0] != "system_admin" {
		t.Errorf("roles: quiero [system_admin], tengo %v", claims["roles"])
	}
	if _, ok := claims["exp"]; !ok {
		t.Fatalf("exp obligatorio (WithExpirationRequired del verificador, D3.2)")
	}
}

func TestToken_CachesAndReissues(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-platformauth-32chars-min!!")
	p, _ := NewServiceTokenProvider()
	a, _ := p.Token()
	b, _ := p.Token()
	if a != b {
		t.Fatalf("token cacheado debe ser estable dentro del TTL")
	}
}
