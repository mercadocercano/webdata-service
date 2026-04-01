package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	srccontroller "github.com/mercadocercano/webdata-service/src/source/infrastructure/controller"
	scrapecontroller "github.com/mercadocercano/webdata-service/src/scraping/infrastructure/controller"
	prdcontroller "github.com/mercadocercano/webdata-service/src/product/infrastructure/controller"
	statscontroller "github.com/mercadocercano/webdata-service/src/stats/infrastructure/controller"
	sharedmw "github.com/mercadocercano/webdata-service/src/shared/middleware"
)

func NewRouter(
	sourceCtrl *srccontroller.SourceController,
	jobCtrl *scrapecontroller.JobController,
	productCtrl *prdcontroller.ProductController,
	statsCtrl *statscontroller.StatsController,
) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(sharedmw.TenantMiddleware)

		// Sources
		r.Route("/sources", func(r chi.Router) {
			r.Get("/", sourceCtrl.List)
			r.Post("/", sourceCtrl.Create)
			r.Get("/{id}", sourceCtrl.Get)
			r.Patch("/{id}", sourceCtrl.Update)
			r.Delete("/{id}", sourceCtrl.Delete)
			r.Post("/{id}/trigger", sourceCtrl.Trigger)
		})

		// Jobs
		r.Route("/jobs", func(r chi.Router) {
			r.Get("/", jobCtrl.List)
			r.Get("/{id}", jobCtrl.Get)
			r.Post("/{id}/cancel", jobCtrl.Cancel)
			r.Post("/{id}/retry", jobCtrl.Retry)
		})

		// Products
		r.Route("/products", func(r chi.Router) {
			r.Get("/", productCtrl.List)
			r.Get("/{id}", productCtrl.Get)
			r.Get("/{id}/price-history", productCtrl.PriceHistory)
		})

		// Stats
		r.Get("/stats", statsCtrl.GetStats)
		r.Get("/stats/sources", statsCtrl.GetSourceStats)
	})

	return r
}
