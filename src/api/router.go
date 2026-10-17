package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	enrichcontroller "github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/controller"
	productcontroller "github.com/mercadocercano/webdata-service/src/product/infrastructure/controller"
	scrapingcontroller "github.com/mercadocercano/webdata-service/src/scraping/infrastructure/controller"
	"github.com/mercadocercano/webdata-service/src/shared/middleware"
	sourcecontroller "github.com/mercadocercano/webdata-service/src/source/infrastructure/controller"
	statscontroller "github.com/mercadocercano/webdata-service/src/stats/infrastructure/controller"
)

func NewRouter(
	sourceCtrl *sourcecontroller.SourceController,
	jobCtrl *scrapingcontroller.JobController,
	productCtrl *productcontroller.ProductController,
	statsCtrl *statscontroller.StatsController,
	btProxyCtrl *productcontroller.BusinessTypeProxyController,
	enrichCtrl *enrichcontroller.EnrichmentHandler,
) http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		middleware.JSONResponse(w, http.StatusOK, map[string]string{"status": "ok", "service": "webdata-service"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		// AuthMiddleware is the ONLY identity path: HS256 JWT verified in-service,
		// `exp` required, roles derived from the `roles` claim (PLAT-E39 T9c1/D9b).
		// The legacy `X-API-Key` + `X-User-Role` bypass with its baked fallback was
		// removed (D9b). TenantMiddleware is applied per-group below, NOT globally,
		// so the admin surface can gate on role without first demanding X-Tenant-ID
		// (a tenant_admin hitting /enrichment/* must get 403, not 400 for a missing
		// tenant header).
		r.Use(middleware.AuthMiddleware)

		// ─── Platform / admin surface ─────────────────────────────────────────
		// system_admin required, deny-by-default (D3.2 — no RLS backstop on the
		// platform tables these routes drive). No X-Tenant-ID: platform callers carry
		// no tenant by design; the enrichment batch reads cross-tenant via the explicit
		// platform read path (T9c2), not via a sentinel tenant_id in context.
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRoleHTTP("system_admin"))

			// Enrichment batch / status / reject.
			r.Route("/enrichment", func(r chi.Router) {
				r.Post("/run", enrichCtrl.RunBatch)
				r.Get("/status", enrichCtrl.GetStatus)
				r.Post("/reject", enrichCtrl.RejectImage)
			})

			// Triggers cross-tenant enrichment via the platform read path. Mounted as a
			// standalone admin route (NOT inside the tenant /products group) so the role
			// gate runs without a tenant requirement; chi matches this static path before
			// the /products/{id} param route below. A tenant caller no longer reaches the
			// platform read path (D9b/D9a).
			r.Post("/products/enrich-from-global-catalog", productCtrl.EnrichFromGlobalCatalog)
		})

		// ─── Tenant surface ───────────────────────────────────────────────────
		// Tenant-scoped CRUD: X-Tenant-ID required (fail-closed). system_admin callers
		// skip the requirement but get no tenant in context (handlers that read
		// TenantIDFromContext then report a missing tenant — intended: an admin listing
		// tenant data must name the tenant).
		r.Group(func(r chi.Router) {
			r.Use(middleware.TenantMiddleware)

			// Sources
			r.Route("/sources", func(r chi.Router) {
				r.Get("/", sourceCtrl.ListSources)
				r.Post("/", sourceCtrl.CreateSource)
				r.Get("/{id}", sourceCtrl.GetSource)
				r.Patch("/{id}", sourceCtrl.UpdateSource)
				r.Delete("/{id}", sourceCtrl.DeleteSource)
				r.Post("/{id}/trigger", sourceCtrl.TriggerScrape)
			})

			// Jobs
			r.Route("/jobs", func(r chi.Router) {
				r.Get("/", jobCtrl.ListJobs)
				r.Get("/{id}", jobCtrl.GetJob)
				r.Post("/{id}/cancel", jobCtrl.CancelJob)
				r.Post("/{id}/retry", jobCtrl.RetryJob)
			})

			// Products (tenant CRUD — the platform enrich-from-global-catalog route
			// lives in the admin group above).
			r.Route("/products", func(r chi.Router) {
				r.Get("/", productCtrl.ListProducts)
				r.Get("/filters", productCtrl.GetProductFilters)
				r.Post("/sync-to-pim", productCtrl.SyncToPIM)
				r.Post("/bulk-delete", productCtrl.BulkDeleteProducts)
				r.Post("/bulk-assign-business-type", productCtrl.BulkAssignBusinessType)
				r.Post("/auto-assign-business-types", productCtrl.AutoMatchBusinessTypes)
				r.Get("/{id}", productCtrl.GetProduct)
				r.Delete("/{id}", productCtrl.DeleteProduct)
				r.Patch("/{id}", productCtrl.PatchProduct)
				r.Get("/{id}/price-history", productCtrl.GetPriceHistory)
				r.Put("/{id}/business-types", productCtrl.AssignBusinessTypes)
				r.Delete("/{id}/business-types/{code}", productCtrl.RemoveBusinessType)
			})

			// Business Types (proxy to pim-service)
			r.Get("/business-types", btProxyCtrl.ListBusinessTypes)

			// Stats
			r.Route("/stats", func(r chi.Router) {
				r.Get("/", statsCtrl.GetStats)
				r.Get("/sources", statsCtrl.GetSourceStats)
				r.Get("/pipeline", statsCtrl.GetPipelineStatus)
			})
		})
	})

	return r
}
