package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	webdata "github.com/mercadocercano/webdata-service"
	"github.com/mercadocercano/webdata-service/src/api"
	enrichconfig "github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/config"
	productconfig "github.com/mercadocercano/webdata-service/src/product/infrastructure/config"
	productcontroller "github.com/mercadocercano/webdata-service/src/product/infrastructure/controller"
	"github.com/mercadocercano/webdata-service/src/scraping/infrastructure/adapter"
	scrapingconfig "github.com/mercadocercano/webdata-service/src/scraping/infrastructure/config"
	"github.com/mercadocercano/webdata-service/src/scraping/infrastructure/scheduler"
	"github.com/mercadocercano/webdata-service/src/shared/database"
	sourceconfig "github.com/mercadocercano/webdata-service/src/source/infrastructure/config"
	statsconfig "github.com/mercadocercano/webdata-service/src/stats/infrastructure/config"
	webdatalogging "github.com/mercadocercano/webdata-service/src/webdata/infrastructure/logging"

	"github.com/hornosg/go-shared/infrastructure/postgres"
	sharedmigrate "github.com/hornosg/go-shared/migrate"
)

func main() {
	port := getEnv("PORT", "8150")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPass := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "webdata")
	firecrawlKey := getEnv("FIRECRAWL_API_KEY", "")

	if os.Getenv("JWT_SECRET") == "" {
		fmt.Fprintln(os.Stderr, "JWT_SECRET env var is required")
		os.Exit(1)
	}

	db, err := database.NewPostgresDB(dbHost, dbPort, dbUser, dbPass, dbName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Run migrations before starting the server (fail-fast per ADR-001).
	if err := sharedmigrate.RunMigrations(db, webdata.MigrationsFS, dbName); err != nil {
		fmt.Fprintf(os.Stderr, "migrations failed: %v\n", err)
		os.Exit(1)
	}

	postgres.StartPoolMonitor(context.Background(), db, postgres.MonitorOptions{
		Service: "webdata-service",
		DBName:  dbName,
	})

	// Canonical logger (ADR-001)
	webdataLogger := webdatalogging.NewWebdataLogger("webdata-service")

	// DI wiring
	productModule := productconfig.NewProductModule(db, webdataLogger)
	sourceModule := sourceconfig.NewSourceModule(db, nil) // job repo wired after scraping module

	firecrawlAdapter := adapter.NewFirecrawlAdapter(firecrawlKey)
	scrapingModule := scrapingconfig.NewScrapingModule(
		db,
		sourceModule.Repo,
		firecrawlAdapter,
		productModule.UpsertUC,
	)

	// Rewire source module with correct job repo
	sourceModule = sourceconfig.NewSourceModule(db, scrapingModule.Repo)

	// Wire enrichment use case now that source+job repos are ready
	productModule.WireEnrichment(sourceModule.Repo, scrapingModule.Repo)

	statsModule := statsconfig.NewStatsModule(sourceModule.Repo, scrapingModule.Repo, productModule.Repo)
	btProxyCtrl := productcontroller.NewBusinessTypeProxyController()
	enrichModule := enrichconfig.NewEnrichmentModule(db)

	router := api.NewRouter(
		sourceModule.Controller,
		scrapingModule.Controller,
		productModule.Controller,
		statsModule.Controller,
		btProxyCtrl,
		enrichModule.Handler,
	)

	// Scheduler
	sched := scheduler.NewScheduler(sourceModule.Repo, scrapingModule.Repo)
	sched.WithLogger(webdataLogger)
	workerPool := scheduler.NewWorkerPool(scrapingModule.ExecuteUC, 3)
	workerPool.WithLogger(webdataLogger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sched.Start(ctx)
	go workerPool.Start(ctx)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Printf("webdata-service listening on :%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-quit
	fmt.Println("shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
	}
	fmt.Println("stopped")
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
