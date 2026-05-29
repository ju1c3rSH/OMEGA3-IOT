package MiddleWares

import (
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminAuthMiddleware verifies the JWT and ensures the user has any admin-level role.
// Must be placed after JwtAuthMiddleWare in the middleware chain.
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
			c.Abort()
			return
		}

		role, ok := roleVal.(int)
		if !ok {
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Invalid role type in context"))
			c.Abort()
			return
		}

		r := model.Role(role)
		if !r.IsAdmin() {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Admin access required"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission returns middleware that enforces a specific permission.
// The user's role is read from the Gin context (set by JwtAuthMiddleWare).
//
// Usage:
//
//	adminGroup.GET("/users", RequirePermission(model.PermUserView), handler.ListUsers)
func RequirePermission(perm model.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
			c.Abort()
			return
		}

		role, ok := roleVal.(int)
		if !ok {
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Invalid role type in context"))
			c.Abort()
			return
		}

		r := model.Role(role)
		if err := r.RequirePermission(perm); err != nil {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(http.StatusForbidden, "Access denied", err.Error()))
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetRole extracts the user's role from the Gin context.
// Returns RoleNormal if not found or invalid.
func GetRole(c *gin.Context) model.Role {
	roleVal, exists := c.Get("role")
	if !exists {
		return model.RoleNormal
	}
	role, ok := roleVal.(int)
	if !ok {
		return model.RoleNormal
	}
	return model.Role(role)
}
