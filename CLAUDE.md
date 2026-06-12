# CLAUDE.md - webdata-service

Servicio Go para scraping de productos usando Firecrawl. Reemplaza al scraper-service (Python/MongoDB).

**Puerto**: 8150 | **Stack**: Go 1.24 + PostgreSQL + Firecrawl API

Hablame siempre en español.

## Arquitectura

Hexagonal estricta:
```
src/{domain}/domain/{entity,value_object,port,exception}/
src/{domain}/application/{usecase,request,response}/
src/{domain}/infrastructure/{persistence,adapter,controller,config}/
src/shared/{middleware,database}/
src/api/router.go
cmd/api/main.go
```

Dominios: `source`, `scraping`, `product`, `stats`

## Comandos

```bash
go run ./cmd/api/main.go   # Desarrollo local
go build ./...             # Compilar
go test ./...              # Tests
go test ./src/.../domain/... # Solo domain tests
```

## Reglas críticas

- SIEMPRE tenant_id en todas las entidades y queries
- Headers: Authorization: Bearer <jwt>, X-Tenant-ID: <uuid>
- Solo PostgreSQL, nunca MongoDB
- GOPRIVATE=github.com/mercadocercano/* para libs privadas
- Firecrawl usa FIRECRAWL_API_KEY del entorno
- Puerto 8150, ruta Kong: /webdata/api/v1/...

## Modelo de datos

Tablas: `webdata_sources`, `webdata_scraping_jobs`, `webdata_products`, `webdata_price_history`
Ver migrations/ para el schema exacto.

## Integración Firecrawl

- `extract`: para ecommerce con schema LLM-powered
- `scrape`: para páginas simples estáticas
- `crawl`: para catálogos paginados
- Rate limiting + retry interno en FirecrawlAdapter

## Memoria persistente (Engram)

Tenés acceso a memoria persistente entre sesiones vía las herramientas MCP de Engram (`mem_save`, `mem_search`, `mem_context`, etc.). Proyecto: **`mercado-cercano`** — la memoria es compartida con el resto del ecosistema (IAM, PIM).

**Cuándo guardar** — sin esperar que te lo pidan:
- Al resolver un bug no trivial: síntoma, causa raíz, fix aplicado.
- Al tomar una decisión de diseño: qué se decidió y por qué.
- Al descubrir un patrón o convención del proyecto que no está documentada.
- Al completar una feature o refactor significativo: qué cambió y dónde.

**Cuándo buscar** — antes de empezar cualquier tarea:
- `mem_context` al inicio de sesión o tras una compaction para recuperar el estado anterior.
- `mem_search` cuando el usuario menciona algo que puede tener historial ("el bug de scraping", "la migración de la semana pasada").

**Al cerrar sesión**: llamar `mem_session_summary` para dejar un resumen recuperable.
