# webdata-service

Servicio Go de adquisición y normalización de datos de productos para una plataforma SaaS multi-tenant. Descubre catálogos de fuentes externas, los scrapea con [Firecrawl](https://firecrawl.dev), los normaliza, enriquece y los sincroniza hacia el catálogo del marketplace (PIM).

```
module github.com/mercadocercano/webdata-service   ·   Go 1.25   ·   puerto 8150
```

Reemplaza a un scraper previo en Python/MongoDB por una implementación en Go bajo **arquitectura hexagonal estricta**, multi-tenant de punta a punta y con persistencia única en PostgreSQL. Es parte del ecosistema [mercado-cercano](https://github.com/mercadocercano).

## Qué resuelve

Poblar el catálogo de un marketplace de comercios de barrio exige traer miles de productos desde fuentes heterogéneas (sitios de ecommerce, catálogos paginados, páginas estáticas) y dejarlos limpios, deduplicados y clasificados antes de que lleguen al PIM. `webdata-service` encapsula ese pipeline: del HTML crudo a un producto normalizado, con historial de precios y trazabilidad por fuente.

## Pipeline

```
source  →  scraping  →  product  →  enrichment  →  sync-to-pim
fuente     job de       producto    clasificación   alta en el
+ schedule extracción   normalizado + dedup global  catálogo PIM
```

- **source** — define una fuente (URL, método de scraping, schema de extracción, prompt LLM) y su agenda de re-scrapeo.
- **scraping** — ejecuta jobs contra Firecrawl con rate limiting y retry; soporta `extract` (ecommerce con schema LLM-powered), `scrape` (páginas estáticas) y `crawl` (catálogos paginados).
- **product** — normaliza, mantiene `webdata_price_history`, asigna business types y enriquece contra el catálogo global.
- **enrichment** — corrida de enriquecimiento con estado y posibilidad de rechazo manual.
- **stats** — métricas del pipeline por fuente y agregadas.

Un **scheduler** (`robfig/cron`) dispara los re-scrapeos según la agenda de cada fuente; las fuentes con fallos consecutivos se desactivan automáticamente.

## Arquitectura

Hexagonal por dominio. Cada bounded context aísla `domain` / `application` / `infrastructure`:

```
src/{source,scraping,product,enrichment,stats}/
  domain/         entities, value objects, ports, exceptions
  application/    use cases, request/response
  infrastructure/ persistence, adapters, controllers, config
src/shared/       middleware, database, httputil
src/api/router.go
cmd/api/main.go
```

El dominio no conoce a Firecrawl ni a Postgres: ambos entran como **adapters** detrás de puertos (`FirecrawlAdapter`, repositorios). El kernel transversal —pool de Postgres con monitoreo de saturación, logging canónico (ADR-001)— se consume desde [`go-shared`](https://github.com/hornosg/go-shared).

## Multi-tenancy

`tenant_id` es obligatorio en **toda** entidad y query. Las requests viajan con:

```
Authorization: Bearer <jwt>
X-Tenant-ID:   <uuid>
```

## API

Base: `/api/v1` (vía Kong: `/webdata/api/v1/...`). OpenAPI en [`api-docs/openapi.yaml`](api-docs/openapi.yaml).

| Recurso | Endpoints destacados |
|---------|----------------------|
| Fuentes | `GET/POST /sources`, `POST /sources/{id}/trigger` |
| Jobs | `GET /jobs`, `POST /jobs/{id}/cancel`, `POST /jobs/{id}/retry` |
| Productos | `GET /products`, `POST /products/sync-to-pim`, `/bulk-delete`, `/auto-assign-business-types`, `/enrich-from-global-catalog` |
| Enriquecimiento | `POST /enrichment/run`, `GET /enrichment/status`, `POST /enrichment/reject` |
| Stats | `GET /stats`, `/stats/sources`, `/stats/pipeline` |

## Modelo de datos

PostgreSQL, migraciones evolutivas versionadas (`migrations/`):
`webdata_sources` · `webdata_scraping_jobs` · `webdata_products` · `webdata_price_history`.

## Desarrollo

```bash
go run ./cmd/api/main.go   # local (:8150)
go build ./...             # compilar
go test ./...              # tests (testcontainers para integración)
```

Variables de entorno en [`.env.example`](.env.example) — Postgres, `FIRECRAWL_API_KEY`, `JWT_SECRET`, `PORT`. `JWT_SECRET` es obligatorio o el servicio no arranca. Para resolver las libs privadas: `GOPRIVATE=github.com/mercadocercano/*`.

---

Parte del ecosistema [mercado-cercano](https://github.com/mercadocercano) · Licencia MIT
