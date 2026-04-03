package database

import "github.com/google/uuid"

var GlobalTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// TenantWhereClause returns the WHERE clause and args for tenant filtering.
// For global tenant (admin), it returns "WHERE 1=1" with no args, matching all tenants.
// For regular tenants, it returns "WHERE tenant_id=$1" with the tenant UUID.
func TenantWhereClause(tenantID uuid.UUID) (where string, args []interface{}, nextArgIdx int) {
	if tenantID == GlobalTenantID {
		return "WHERE 1=1", nil, 1
	}
	return "WHERE tenant_id=$1", []interface{}{tenantID}, 2
}
