package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mercadocercano/webdata-service/src/api"
	"github.com/mercadocercano/webdata-service/src/scraping/infrastructure/adapter"
	scrapingconfig "github.com/mercadocercano/webdata-service/src/scraping/infrastructure/config"
	"github.com/mercadocercano/webdata-service/src/scraping/infrastructure/scheduler"
	productconfig "github.com/mercadocercano/webdata-service/src/product/infrastructure/config"
	productcontroller "github.com/mercadocercano/webdata-service/src/product/infrastructure/controller"
	sourceconfig "github.com/mercadocercano/webdata-service/src/source/infrastructure/config"
	statsconfig "github.com/mercadocercano/webdata-service/src/stats/infrastructure/config"
	"github.com/mercadocercano/webdata-service/src/shared/database"
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

	connStr := database.BuildConnString(dbHost, dbPort, dbUser, dbPass, dbName)
	db, err := database.NewPostgresDB(connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// DI wiring
	productModule := productconfig.NewProductModule(db)
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

	statsModule := statsconfig.NewStatsModule(sourceModule.Repo, scrapingModule.Repo, productModule.Repo)
	btProxyCtrl := productcontroller.NewBusinessTypeProxyController()

	router := api.NewRouter(
		sourceModule.Controller,
		scrapingModule.Controller,
		productModule.Controller,
		statsModule.Controller,
		btProxyCtrl,
	)

	// Scheduler
	sched := scheduler.NewScheduler(sourceModule.Repo, scrapingModule.Repo)
	workerPool := scheduler.NewWorkerPool(scrapingModule.ExecuteUC, 3)

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
