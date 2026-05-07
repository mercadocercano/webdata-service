package entity

import "time"

type Source string

const (
	SourceML         Source = "ml"
	SourceOFF        Source = "off"
	SourceFirecrawl  Source = "firecrawl"
	SourceManual     Source = "manual"
	SourcePerplexity Source = "perplexity"
	SourceDALLE      Source = "dalle"
)

type Enhancer string

const (
	EnhancerPassthrough Enhancer = "passthrough"
	EnhancerReplicate   Enhancer = "replicate"
	EnhancerDALLE       Enhancer = "dalle"
)

type ImageResolution struct {
	GTIN         string
	EAN          string
	ProductID    string
	Source       Source
	RawURL       string
	CDNURL       string
	CDNKey       string
	Enhancer     Enhancer
	CostCents    int
	QualityScore int
	RejectedURLs []string
	ResolvedAt   time.Time
	UpdatedAt    time.Time
}
