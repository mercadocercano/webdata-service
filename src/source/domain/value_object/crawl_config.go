package value_object

import "encoding/json"

type CrawlConfig struct {
	MaxDepth    int      `json:"max_depth,omitempty"`
	URLPatterns []string `json:"url_patterns,omitempty"`
	MaxPages    int      `json:"max_pages,omitempty"`
	IgnoreRobots bool   `json:"ignore_robots,omitempty"`
}

func NewCrawlConfig(maxDepth int, urlPatterns []string) CrawlConfig {
	return CrawlConfig{
		MaxDepth:    maxDepth,
		URLPatterns: urlPatterns,
	}
}

func CrawlConfigFromJSON(raw json.RawMessage) (CrawlConfig, error) {
	var cfg CrawlConfig
	if len(raw) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return CrawlConfig{}, err
	}
	return cfg, nil
}

func (c CrawlConfig) ToJSON() (json.RawMessage, error) {
	return json.Marshal(c)
}
