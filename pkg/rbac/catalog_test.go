package rbac

import "testing"

func TestCatalog_coversAllPermissionConstants(t *testing.T) {
	want := map[Permission]struct{}{
		PermUsersList: {}, PermUsersRead: {}, PermUsersActivate: {},
		PermUsersDelete: {}, PermUsersRestore: {}, PermUsersRoleChange: {},
		PermUsersEmailAny: {}, PermUploadsCreate: {}, PermRBACManage: {},
		PermNotificationsSend: {}, PermNotificationsSettings: {},
		PermPaymentsCharge: {}, PermPaymentsList: {}, PermAuditList: {},
		PermContactsList: {},
	}
	for _, def := range Catalog() {
		delete(want, def.Permission)
		if def.Description == "" {
			t.Fatalf("catalog entry %q missing description", def.Permission)
		}
	}
	if len(want) > 0 {
		t.Fatalf("catalog missing permissions: %v", want)
	}
}
