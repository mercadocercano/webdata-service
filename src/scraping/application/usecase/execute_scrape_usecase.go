package usecase

import (
	"context"
	"fmt"

	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
	productusecase "github.com/mercadocercano/webdata-service/src/product/application/usecase"
)

type ExecuteScrapingUseCase struct {
	jobRepo    scrapingport.ScrapingJobRepository
	sourceRepo sourceport.SourceRepository
	scraper    scrapingport.ScraperPort
	upsertUC   *productusecase.UpsertProductsUseCase
}

func NewExecuteScrapingUseCase(
	jobRepo scrapingport.ScrapingJobRepository,
	sourceRepo sourceport.SourceRepository,
	scraper scrapingport.ScraperPort,
	upsertUC *productusecase.UpsertProductsUseCase,
) *ExecuteScrapingUseCase {
	return &ExecuteScrapingUseCase{
		jobRepo:    jobRepo,
		sourceRepo: sourceRepo,
		scraper:    scraper,
		upsertUC:   upsertUC,
	}
}

func (uc *ExecuteScrapingUseCase) ClaimJob(ctx context.Context) (*entity.ScrapingJob, error) {
	return uc.jobRepo.ClaimPendingJob(ctx)
}

func (uc *ExecuteScrapingUseCase) Execute(ctx context.Context, job *entity.ScrapingJob) error {
	if err := job.Start(); err != nil {
		return fmt.Errorf("starting job: %w", err)
	}
	if err := uc.jobRepo.Update(ctx, job); err != nil {
		return fmt.Errorf("updating job to running: %w", err)
	}

	source, err := uc.sourceRepo.FindByID(ctx, job.TenantID, job.SourceID)
	if err != nil {
		_ = job.Fail("source not found")
		_ = uc.jobRepo.Update(ctx, job)
		return err
	}

	rawProducts, err := uc.scraper.Extract(ctx, source.BaseURL, source.ExtractionSchema.Raw(), scrapingport.ExtractOptions{})
	if err != nil {
		reason := fmt.Sprintf("extract failed: %v", err)
		_ = job.Fail(reason)
		_ = uc.jobRepo.Update(ctx, job)
		source.RecordFailure(reason)
		_ = uc.sourceRepo.Update(ctx, source)
		return err
	}

	created, updated, upsertErr := uc.upsertUC.ExecuteDetailed(ctx, job.TenantID, job.SourceID, &job.ID, rawProducts)

	found := len(rawProducts)
	saved := created + updated
	if upsertErr != nil {
		_ = job.Fail(upsertErr.Error())
		source.RecordFailure(upsertErr.Error())
	} else {
		_ = job.Complete(found, saved)
		source.RecordSuccess()
	}

	_ = uc.jobRepo.Update(ctx, job)
	_ = uc.sourceRepo.Update(ctx, source)
	return upsertErr
}
