package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	sourcecontroller "github.com/mercadocercano/webdata-service/src/source/infrastructure/controller"
	scrapingcontroller "github.com/mercadocercano/webdata-service/src/scraping/infrastructure/controller"
	productcontroller "github.com/mercadocercano/webdata-service/src/product/infrastructure/controller"
	statscontroller "github.com/mercadocercano/webdata-service/src/stats/infrastructure/controller"
	"github.com/mercadocercano/webdata-service/src/shared/middleware"
)

func NewRouter(
	sourceCtrl *sourcecontroller.SourceController,
	jobCtrl *scrapingcontroller.JobController,
	productCtrl *productcontroller.ProductController,
	statsCtrl *statscontroller.StatsController,
	btProxyCtrl *productcontroller.BusinessTypeProxyController,
) http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		middleware.JSONResponse(w, http.StatusOK, map[string]string{"status": "ok", "service": "webdata-service"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
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

		// Products
		r.Route("/products", func(r chi.Router) {
			r.Get("/", productCtrl.ListProducts)
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
		})
	})

	return r
}
