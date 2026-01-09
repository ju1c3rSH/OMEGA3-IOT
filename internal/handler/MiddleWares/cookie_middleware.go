package MiddleWares

import (
	"OMEGA3-IOT/internal/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

func CookieAuthMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		var tokenString string
		//var tokenSource string
		authInCookie, err := context.Request.Cookie("Authorization")
		if err == nil && authInCookie != nil {
			tokenString = authInCookie.Value
		} else {
			context.JSON(http.StatusUnauthorized, gin.H{"error": "No Authorization in cookie"})
			context.Abort()
			return
		}
		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = tokenString[7:]
		}
		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			context.Abort()
			return

		}
		if claims.ExpiresAt < time.Now().Unix() {
			context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			context.Abort()
			return
		}
		context.Set("username", claims.UserName)
		context.Set("role", claims.Role)
		context.Set("user_uuid", claims.UUID)
		context.Set("ExpiresAt", claims.ExpiresAt)
		context.Next()
	}
}

//已经迁移到JwtAuthMiddleWare()
