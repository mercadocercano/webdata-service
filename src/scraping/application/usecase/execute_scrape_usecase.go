package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
)

// ProductUpserter abstracts the product upsert operation to avoid circular imports.
type ProductUpserter interface {
	Execute(ctx context.Context, tenantID, sourceID uuid.UUID, jobID *uuid.UUID, products []scrapeport.RawProduct) (int, error)
}

type ExecuteScrapeUseCase struct {
	sourceRepo sourceport.SourceRepository
	jobRepo    scrapeport.ScrapingJobRepository
	scraper    scrapeport.ScraperPort
	upserter   ProductUpserter
}

func NewExecuteScrapeUseCase(
	sourceRepo sourceport.SourceRepository,
	jobRepo scrapeport.ScrapingJobRepository,
	scraper scrapeport.ScraperPort,
	upserter ProductUpserter,
) *ExecuteScrapeUseCase {
	return &ExecuteScrapeUseCase{
		sourceRepo: sourceRepo,
		jobRepo:    jobRepo,
		scraper:    scraper,
		upserter:   upserter,
	}
}

func (uc *ExecuteScrapeUseCase) Execute(ctx context.Context) error {
	job, err := uc.jobRepo.ClaimPendingJob(ctx)
	if err != nil {
		return nil // no pending jobs — not an error
	}

	if err := job.Start(); err != nil {
		return fmt.Errorf("starting job %s: %w", job.ID, err)
	}
	if err := uc.jobRepo.Update(ctx, job); err != nil {
		return fmt.Errorf("updating job to running: %w", err)
	}

	source, err := uc.sourceRepo.FindByID(ctx, job.TenantID, job.SourceID)
	if err != nil {
		_ = job.Fail("source not found")
		_ = uc.jobRepo.Update(ctx, job)
		return nil
	}

	schemaBytes := source.ExtractionSchema.Raw()
	if schemaBytes == nil {
		schemaBytes = []byte("{}")
	}
	var schema json.RawMessage = schemaBytes

	products, err := uc.scraper.Extract(ctx, source.BaseURL, schema, scrapeport.ExtractOptions{})
	if err != nil {
		_ = job.Fail(err.Error())
		_ = uc.jobRepo.Update(ctx, job)
		source.RecordFailure(err.Error())
		_ = uc.sourceRepo.Update(ctx, source)
		return nil
	}

	saved, err := uc.upserter.Execute(ctx, job.TenantID, job.SourceID, &job.ID, products)
	if err != nil {
		_ = job.Fail(err.Error())
		_ = uc.jobRepo.Update(ctx, job)
		return nil
	}

	_ = job.Complete(len(products), saved)
	_ = uc.jobRepo.Update(ctx, job)
	source.RecordSuccess()
	_ = uc.sourceRepo.Update(ctx, source)

	return nil
}
