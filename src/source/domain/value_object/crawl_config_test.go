package value_object_test

import (
	"encoding/json"
	"testing"

	"github.com/mercadocercano/webdata-service/src/source/domain/value_object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrawlConfig_EffectivePageParam_DefaultsToPage(t *testing.T) {
	cfg := value_object.CrawlConfig{MaxPages: 5}
	assert.Equal(t, "page", cfg.EffectivePageParam())
}

func TestCrawlConfig_EffectivePageParam_UsesCustomParam(t *testing.T) {
	cfg := value_object.CrawlConfig{MaxPages: 5, PageParam: "p"}
	assert.Equal(t, "p", cfg.EffectivePageParam())
}

func TestCrawlConfig_FromJSON_ParsesPageParam(t *testing.T) {
	raw := json.RawMessage(`{"max_pages": 3, "page_param": "p"}`)
	cfg, err := value_object.CrawlConfigFromJSON(raw)

	require.NoError(t, err)
	assert.Equal(t, 3, cfg.MaxPages)
	assert.Equal(t, "p", cfg.PageParam)
}

func TestCrawlConfig_ToJSON_IncludesPageParam(t *testing.T) {
	cfg := value_object.CrawlConfig{MaxPages: 5, PageParam: "p"}
	raw, err := cfg.ToJSON()

	require.NoError(t, err)

	var parsed map[string]interface{}
	_ = json.Unmarshal(raw, &parsed)
	assert.Equal(t, float64(5), parsed["max_pages"])
	assert.Equal(t, "p", parsed["page_param"])
}
