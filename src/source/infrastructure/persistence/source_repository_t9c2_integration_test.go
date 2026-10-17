//go:build integration

// TEST-ID: T9 / T10 (PLAT-E39 T9c2)
// Read-only integration test for the platform read path that replaces the GlobalTenantID
// sentinel. It derives a real tenant/category/source from the migrated webdata_db at
// runtime (no hardcoded UUIDs, no mutation — safe against the shared lab DB), then:
//
//   T9  — FindAllActiveByCategoryPlatform(cat) returns exactly the active sources for that
//         category with NO tenant filter (parity vs `WHERE is_active=true AND category=$1`).
//   T10 — the tenant-scoped callers stay fail-closed now that the sentinel is gone:
//         FindAll(uuid.Nil, ...) returns 0 (was "WHERE 1=1" → all tenants),
//         FindByID(uuid.Nil, realID) returns not-found (was cross-tenant),
//         and the real tenant's FindAll/FindByID still scope by tenant_id.
//
// Run: WEBDATA_TEST_DB_URL=... go test -tags=integration ./src/source/infrastructure/persistence/...
// (defaults to localhost:5432/postgres/postgres/webdata_db).
package persistence_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/mercadocercano/webdata-service/src/source/infrastructure/persistence"
	"github.com/mercadocercano/webdata-service/src/source/domain/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func t9c2DB(t *testing.T) *sql.DB {
	t.Helper()
	connStr := os.Getenv("WEBDATA_TEST_DB_URL")
	if connStr == "" {
		connStr = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			getEnv("DB_HOST", "localhost"),
			getEnv("DB_PORT", "5432"),
			getEnv("DB_USER", "postgres"),
			getEnv("DB_PASSWORD", "postgres"),
			getEnv("DB_NAME", "webdata_db"),
		)
	}
	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err, "open test DB")
	require.NoError(t, db.Ping(), "ping test DB")
	t.Cleanup(func() { db.Close() })
	return db
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// A real (tenantID, category, sourceID) tuple drawn from webdata_sources so the test is
// independent of specific seed data.
func realTenantCategorySource(t *testing.T, db *sql.DB) (tenantID, category string, sourceID uuid.UUID) {
	t.Helper()
	row := db.QueryRow(`
		SELECT tenant_id::text, category, id::text
		FROM webdata_sources
		WHERE is_active = true
		ORDER BY created_at DESC
		LIMIT 1`)
	var tid, cat, sid string
	require.NoError(t, row.Scan(&tid, &cat, &sid), "need at least one active source in webdata_db")
	parsed, err := uuid.Parse(sid)
	require.NoError(t, err)
	return tid, cat, parsed
}

// TEST-ID: T9 — platform read path returns active sources by category with no tenant filter.
func TestFindAllActiveByCategoryPlatform_ParityNoTenantFilter(t *testing.T) {
	db := t9c2DB(t)
	_, category, _ := realTenantCategorySource(t, db)
	repo := persistence.NewPostgresSourceRepository(db)

	got, err := repo.FindAllActiveByCategoryPlatform(context.Background(), category)
	require.NoError(t, err)

	// Direct SQL with NO tenant clause — the operative check that the method doesn't filter.
	var wantIDs []string
	rows, err := db.Query(`SELECT id::text FROM webdata_sources WHERE is_active=true AND category=$1`, category)
	require.NoError(t, err)
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		wantIDs = append(wantIDs, id)
	}
	require.NoError(t, rows.Close(), "close want rows")

	gotIDs := make([]string, 0, len(got))
	for _, s := range got {
		gotIDs = append(gotIDs, s.ID.String())
		assert.True(t, s.IsActive, "platform read must only return active sources")
		assert.Equal(t, category, s.Category, "platform read must scope by category")
	}
	assert.ElementsMatch(t, wantIDs, gotIDs, "FindAllActiveByCategoryPlatform must equal the unfiltered SQL")
	assert.Equal(t, len(wantIDs), len(got), "row count parity with direct SQL")
}

// TEST-ID: T10 — tenant-scoped FindAll stays fail-closed; the sentinel (uuid.Nil) path is gone.
func TestFindAll_TenantScopedFailClosed_SentinelGone(t *testing.T) {
	db := t9c2DB(t)
	tenantIDStr, category, _ := realTenantCategorySource(t, db)
	tenantID := uuid.MustParse(tenantIDStr)
	repo := persistence.NewPostgresSourceRepository(db)
	isActive := true

	t.Run("real tenant scopes by tenant_id", func(t *testing.T) {
		got, total, err := repo.FindAll(context.Background(), tenantID, port.SourceFilter{IsActive: &isActive, Category: category})
		require.NoError(t, err)
		for _, s := range got {
			assert.Equal(t, tenantID, s.TenantID, "FindAll must scope by the real tenant_id")
		}
		// Cross-check against SQL scoped by the same tenant.
		var sqlCount int
		require.NoError(t, db.QueryRow(
			`SELECT count(*) FROM webdata_sources WHERE tenant_id=$1 AND is_active=true AND category=$2`,
			tenantID, category).Scan(&sqlCount))
		assert.Equal(t, sqlCount, total, "FindAll total must match tenant-scoped SQL")
	})

	t.Run("sentinel uuid.Nil returns 0 — was WHERE 1=1 (all tenants)", func(t *testing.T) {
		// The old TenantWhereClause(GlobalTenantID) produced "WHERE 1=1"; with the sentinel
		// removed it now produces "WHERE tenant_id=$1" with uuid.Nil, which matches no row.
		got, total, err := repo.FindAll(context.Background(), uuid.Nil, port.SourceFilter{IsActive: &isActive})
		require.NoError(t, err)
		assert.Empty(t, got, "uuid.Nil must not bypass tenant scoping (sentinel removed)")
		assert.Equal(t, 0, total)
	})
}

// TEST-ID: T10 — tenant-scoped FindByID stays fail-closed; the sentinel branch is gone.
func TestFindByID_TenantScopedFailClosed_SentinelGone(t *testing.T) {
	db := t9c2DB(t)
	tenantIDStr, _, sourceID := realTenantCategorySource(t, db)
	tenantID := uuid.MustParse(tenantIDStr)
	repo := persistence.NewPostgresSourceRepository(db)

	t.Run("real tenant finds its source", func(t *testing.T) {
		s, err := repo.FindByID(context.Background(), tenantID, sourceID)
		require.NoError(t, err)
		assert.Equal(t, sourceID, s.ID)
		assert.Equal(t, tenantID, s.TenantID)
	})

	t.Run("sentinel uuid.Nil does NOT find it cross-tenant", func(t *testing.T) {
		// The old FindByID branch `if tenantID == GlobalTenantID` returned the source ignoring
		// tenant; removed in T9c2, so uuid.Nil (no row has it) must now miss.
		_, err := repo.FindByID(context.Background(), uuid.Nil, sourceID)
		require.Error(t, err, "uuid.Nil must not reach across tenants (sentinel branch removed)")
	})

	t.Run("a different tenant does NOT find another tenant's source", func(t *testing.T) {
		other := uuid.New()
		_, err := repo.FindByID(context.Background(), other, sourceID)
		require.Error(t, err, "cross-tenant FindByID must fail closed")
	})
}