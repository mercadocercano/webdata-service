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
