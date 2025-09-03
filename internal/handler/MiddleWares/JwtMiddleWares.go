package MiddleWares

import (
	"OMEGA3-IOT/internal/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

func JwtAuthMiddleWare() gin.HandlerFunc {
	var tokenString string
	return func(context *gin.Context) {
		authHeader := context.GetHeader("Authorization")
		authInCookie, err := context.Request.Cookie("Authorization")
		if err == nil && authInCookie != nil {
			tokenString = authInCookie.Value
		} else {
			if authHeader == "" {
				context.JSON(http.StatusUnauthorized, gin.H{"error": "Neither Nor Authorization in cookie or Authorization header"})
				context.Abort()
				return
			} else {
				tokenString = authHeader
			}
		}
		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = tokenString[7:]
		}
		claimsFromCookie, err := utils.ParseToken(tokenString)
		if err != nil {
			context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			context.Abort()
			return

		}
		if claimsFromCookie.ExpiresAt < time.Now().Unix() {
			context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			context.Abort()
			return
		} else {
			context.Set("username", claimsFromCookie.UserName)
			context.Set("role", claimsFromCookie.Role)
			context.Set("user_uuid", claimsFromCookie.UUID)
			context.Set("ExpiresAt", claimsFromCookie.ExpiresAt)
			context.Next()
		}
		if authHeader == "" {
			context.JSON(http.StatusUnauthorized, gin.H{"error": "No Authorization header"})
			context.Abort()
			return
		}

		tokenString := authHeader
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = authHeader[7:]
		}
		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			context.Abort()
			return
		}

		// 将用户信息存储到上下文中
		context.Set("username", claims.UserName)
		context.Set("user_uuid", claims.UUID)
		context.Set("role", claims.Role)

		context.Next()
	}

}
