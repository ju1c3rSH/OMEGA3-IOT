package model

import "fmt"

// Role is a type-safe representation of a user's platform role.
// Higher numeric values imply greater privilege.
type Role int

const (
	RoleNormal     Role = 1 // 普通用户（注册默认）
	RoleModerator  Role = 2 // 审核员（只读）
	RoleAdmin      Role = 3 // 管理员
	RoleSuperAdmin Role = 4 // 超级管理员
)

// roleNames maps Role values to human-readable names.
var roleNames = map[Role]string{
	RoleNormal:     "user",
	RoleModerator:  "moderator",
	RoleAdmin:      "admin",
	RoleSuperAdmin: "super_admin",
}

// String implements fmt.Stringer.
func (r Role) String() string {
	if name, ok := roleNames[r]; ok {
		return name
	}
	return fmt.Sprintf("unknown(%d)", int(r))
}

// IsValid checks whether this Role is a known platform role.
func (r Role) IsValid() bool {
	_, ok := roleNames[r]
	return ok
}

// IsAdmin returns true if this role has any admin-level privilege (moderator or above).
func (r Role) IsAdmin() bool {
	return r >= RoleModerator
}

// AtLeast returns true if this role's privilege is >= the given minimum.
func (r Role) AtLeast(min Role) bool {
	return r >= min
}

// ParseRole converts an int to a Role, returning an error if invalid.
func ParseRole(v int) (Role, error) {
	r := Role(v)
	if !r.IsValid() {
		return RoleNormal, fmt.Errorf("invalid role value: %d", v)
	}
	return r, nil
}

// MustParseRole converts an int to a Role, panicking if invalid.
// Use only in tests or initialisation where the value is known-good.
func MustParseRole(v int) Role {
	r, err := ParseRole(v)
	if err != nil {
		panic(err)
	}
	return r
}
