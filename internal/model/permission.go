package model

import "fmt"

// Permission represents a named operation that can be performed on the platform.
// Permissions are grouped by resource (user, device, group, admin, system).
type Permission string

const (
	// User management
	PermUserView   Permission = "user:view"   // view user list / details
	PermUserEdit   Permission = "user:edit"   // edit user info
	PermUserStatus Permission = "user:status" // enable / disable user
	PermUserDelete Permission = "user:delete" // delete user
	PermUserReset  Permission = "user:reset"  // reset password

	// Device management
	PermDeviceView     Permission = "device:view"     // view device list / details
	PermDeviceEdit     Permission = "device:edit"     // edit device info
	PermDeviceDelete   Permission = "device:delete"   // delete device
	PermDeviceTransfer Permission = "device:transfer" // transfer device ownership

	// Group management
	PermGroupView   Permission = "group:view"   // view group list / details
	PermGroupManage Permission = "group:manage" // dissolve group, manage members

	// Admin management (super_admin only)
	PermAdminView   Permission = "admin:view"   // view admin list
	PermAdminManage Permission = "admin:manage" // promote / demote admins

	// System
	PermSystemStats Permission = "system:stats" // view system statistics
	PermSystemLogs  Permission = "system:logs"  // view admin operation logs
)

// rolePermissions defines which permissions each role has.
// This is the single source of truth for authorization.
var rolePermissions = map[Role]map[Permission]bool{
	RoleModerator: {
		PermUserView: true, PermDeviceView: true,
		PermGroupView: true, PermSystemStats: true, PermSystemLogs: true,
	},
	RoleAdmin: {
		// moderator permissions inherited below
		PermUserView: true, PermUserEdit: true, PermUserStatus: true, PermUserReset: true,
		PermDeviceView: true, PermDeviceEdit: true, PermDeviceDelete: true, PermDeviceTransfer: true,
		PermGroupView: true, PermGroupManage: true,
		PermSystemStats: true, PermSystemLogs: true,
	},
	RoleSuperAdmin: {
		// all permissions
		PermUserView: true, PermUserEdit: true, PermUserStatus: true,
		PermUserDelete: true, PermUserReset: true,
		PermDeviceView: true, PermDeviceEdit: true, PermDeviceDelete: true, PermDeviceTransfer: true,
		PermGroupView: true, PermGroupManage: true,
		PermAdminView: true, PermAdminManage: true,
		PermSystemStats: true, PermSystemLogs: true,
	},
}

// HasPermission checks if a role has a specific permission.
func (r Role) HasPermission(perm Permission) bool {
	perms, ok := rolePermissions[r]
	if !ok {
		return false
	}
	return perms[perm]
}

// RequirePermission returns an error if the role does not have the given permission.
func (r Role) RequirePermission(perm Permission) error {
	if !r.HasPermission(perm) {
		return fmt.Errorf("permission denied: role %s does not have permission %s", r, perm)
	}
	return nil
}
