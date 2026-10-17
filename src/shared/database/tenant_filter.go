package database

import "github.com/google/uuid"

// TenantWhereClause returns the WHERE clause and args for tenant filtering.
//
// It is fail-closed: every call yields "WHERE tenant_id=$1" with the caller's real
// tenantID. There is no admin/global branch — the former GlobalTenantID sentinel that
// produced "WHERE 1=1" (matching all tenants) was removed in PLAT-E39 T9c2 (D9a).
// Platform reads that must cross tenants use an explicit, named platform method on the
// repository (e.g. FindAllActiveByCategoryPlatform), never this tenant filter.
func TenantWhereClause(tenantID uuid.UUID) (where string, args []interface{}, nextArgIdx int) {
	return "WHERE tenant_id=$1", []interface{}{tenantID}, 2
}