// Package platformauth emite el JWT de servicio con el que webdata-service se autentica
// contra catalog-service (PLAT-E39 T9b / decisión D8).
//
// D8 (@dev-architect, 2026-08-01) resolvió que webdata es la ingesta de plataforma —un
// caller admin más sobre la superficie ya gateada de catalog-service— y autentica con una
// identidad de servicio del Published Language de identidad, NO con una ruta S2S nueva
// (que sería el séptimo mecanismo ad-hoc que esta épica existe para cerrar). Mientras
// iam-service no emita grants de service-account, el fallback de D8 es un JWT de vida corta
// firmado con JWT_SECRET de vault/env (misma postura HS256 que el resto de la flota), con el
// grupo `system_admin` que catalog-service deriva del claim verificado DENTRO del servicio
// (D3.1). El secreto NUNCA se hornea (enfermedad `marketplace-admin-key-2025`, PLAT-E30
// §D2a): se lee de env y el proceso frena si falta.
//
// catalog-service verifica este token con `go-shared.VerifyPlatformJWT` (HS256, `exp`
// obligatorio, claim `namespace=="mc"`) y luego `RequireRole("system_admin")` sobre el
// claim `roles`. Este helper emite exactamente esos claims.
package platformauth

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ServiceTokenProvider mints (y cachea) el JWT de servicio para llamar a catalog-service.
// Es seguro para uso concurrente.
type ServiceTokenProvider struct {
	secret  string
	ttl     time.Duration
	mu      sync.RWMutex
	cached  string
	expires time.Time
}

// DefaultTTL es la vida del token de servicio. Corta por contrato D8: un token `system_admin`
// sin `exp` sería una llave maestra permanente sobre las 14 tablas de plataforma (D3.2:
// sin backstop RLS), por eso catalog-service exige `exp` (WithExpirationRequired).
const DefaultTTL = 1 * time.Hour

// NewServiceTokenProvider construye un provider desde JWT_SECRET de env. Fail-fast si falta
// el secreto: no hay fallback horneado (D4).
func NewServiceTokenProvider() (*ServiceTokenProvider, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET env var is required (platform service token, D8/D4 — no fallback)")
	}
	return &ServiceTokenProvider{secret: secret, ttl: DefaultTTL}, nil
}

// MustServiceTokenProvider panickea si el secreto falta. Para wiring en main.go donde un
// arranque sin secreto no tiene sentido.
func MustServiceTokenProvider() *ServiceTokenProvider {
	p, err := NewServiceTokenProvider()
	if err != nil {
		panic(err)
	}
	return p
}

// Token devuelve un JWT válido. Re-mint sólo cuando faltan menos de refreshLead en expirar.
const refreshLead = 5 * time.Minute

func (p *ServiceTokenProvider) Token() (string, error) {
	p.mu.RLock()
	if p.cached != "" && time.Until(p.expires) > refreshLead {
		t := p.cached
		p.mu.RUnlock()
		return t, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	// Doble chequeo bajo lock.
	if p.cached != "" && time.Until(p.expires) > refreshLead {
		return p.cached, nil
	}

	now := time.Now()
	exp := now.Add(p.ttl)
	claims := jwt.MapClaims{
		// Rol derivado DENTRO del servicio consumidor (D3.1): catalog-service no confía en
		// headers de rol, sólo en este claim del JWT verificado.
		"roles":     []string{"system_admin"},
		"namespace": "mc",
		"sub":       "webdata-service",
		"iss":       "webdata-service",
		"iat":       now.Unix(),
		"exp":       exp.Unix(), // obligatorio: WithExpirationRequired del verificador.
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(p.secret))
	if err != nil {
		return "", fmt.Errorf("signing platform service token: %w", err)
	}
	p.cached = signed
	p.expires = exp
	return signed, nil
}
