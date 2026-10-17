package database

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TEST-ID: T8 (PLAT-E39 T9c2)
// TenantWhereClause is fail-closed: it must ALWAYS scope by a real tenant_id and never
// produce the old "WHERE 1=1" admin bypass that the GlobalTenantID sentinel generated.
// The zero UUID is covered explicitly — that value used to be the sentinel trigger.
func TestTenantWhereClause_NeverReturnsWhereOneOne(t *testing.T) {
	tenantIDs := []uuid.UUID{
		uuid.Nil,                                                // former sentinel value — must NOT bypass
		uuid.MustParse("00000000-0000-0000-0000-000000000001"), // arbitrary
		uuid.MustParse("123e4567-e89b-12d3-a456-426614174003"), // a real lab tenant
		uuid.New(),                                              // random
	}
	for _, tid := range tenantIDs {
		where, args, nextArgIdx := TenantWhereClause(tid)
		if strings.Contains(where, "1=1") {
			t.Fatalf("TenantWhereClause(%s) returned bypass %q — WHERE 1=1 is forbidden (D9a)", tid, where)
		}
		if !strings.Contains(where, "tenant_id=$1") {
			t.Fatalf("TenantWhereClause(%s) = %q — must scope by tenant_id=$1", tid, where)
		}
		if len(args) != 1 || args[0] != tid {
			t.Fatalf("TenantWhereClause(%s) args = %v — must be [tenantID]", tid, args)
		}
		if nextArgIdx != 2 {
			t.Fatalf("TenantWhereClause(%s) nextArgIdx = %d — must be 2", tid, nextArgIdx)
		}
	}
}