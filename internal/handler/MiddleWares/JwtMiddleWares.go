package MiddleWares

import (
	"OMEGA3-IOT/internal/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func JwtAuthMiddleWare() gin.HandlerFunc {
	return func(context *gin.Context) {
		authHeader := context.GetHeader("Authorization")
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
