package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/stats/application/response"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	productport "github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type GetStatsUseCase struct {
	sourceRepo  sourceport.SourceRepository
	jobRepo     scrapingport.ScrapingJobRepository
	productRepo productport.ProductRepository
}

func NewGetStatsUseCase(
	sourceRepo sourceport.SourceRepository,
	jobRepo scrapingport.ScrapingJobRepository,
	productRepo productport.ProductRepository,
) *GetStatsUseCase {
	return &GetStatsUseCase{
		sourceRepo:  sourceRepo,
		jobRepo:     jobRepo,
		productRepo: productRepo,
	}
}

func (uc *GetStatsUseCase) Execute(ctx context.Context, tenantID uuid.UUID) (response.StatsResponse, error) {
	isActive := true
	allSources, total, err := uc.sourceRepo.FindAll(ctx, tenantID, sourceport.SourceFilter{PageSize: 1})
	if err != nil {
		return response.StatsResponse{}, err
	}
	_ = allSources

	activeSources, _, err := uc.sourceRepo.FindAll(ctx, tenantID, sourceport.SourceFilter{IsActive: &isActive, PageSize: 1})
	if err != nil {
		return response.StatsResponse{}, err
	}
	_ = activeSources

	allJobs, totalJobs, err := uc.jobRepo.FindAll(ctx, tenantID, scrapingport.JobFilter{PageSize: 1})
	if err != nil {
		return response.StatsResponse{}, err
	}
	_ = allJobs

	pendingJobs, totalPending, err := uc.jobRepo.FindAll(ctx, tenantID, scrapingport.JobFilter{Status: "pending", PageSize: 1})
	if err != nil {
		return response.StatsResponse{}, err
	}
	_ = pendingJobs

	runningJobs, totalRunning, err := uc.jobRepo.FindAll(ctx, tenantID, scrapingport.JobFilter{Status: "running", PageSize: 1})
	if err != nil {
		return response.StatsResponse{}, err
	}
	_ = runningJobs

	completedJobs, totalCompleted, err := uc.jobRepo.FindAll(ctx, tenantID, scrapingport.JobFilter{Status: "completed", PageSize: 1})
	if err != nil {
		return response.StatsResponse{}, err
	}
	_ = completedJobs

	failedJobs, totalFailed, err := uc.jobRepo.FindAll(ctx, tenantID, scrapingport.JobFilter{Status: "failed", PageSize: 1})
	if err != nil {
		return response.StatsResponse{}, err
	}
	_ = failedJobs

	allProducts, totalProducts, err := uc.productRepo.FindAll(ctx, tenantID, productport.ProductFilter{PageSize: 1})
	if err != nil {
		return response.StatsResponse{}, err
	}
	_ = allProducts

	activeSourceCount := 0
	activeSrcs, _, _ := uc.sourceRepo.FindAll(ctx, tenantID, sourceport.SourceFilter{IsActive: &isActive, PageSize: 9999})
	activeSourceCount = len(activeSrcs)

	return response.StatsResponse{
		TotalSources:  total,
		ActiveSources: activeSourceCount,
		TotalJobs:     totalJobs,
		PendingJobs:   totalPending,
		RunningJobs:   totalRunning,
		CompletedJobs: totalCompleted,
		FailedJobs:    totalFailed,
		TotalProducts: totalProducts,
	}, nil
}
