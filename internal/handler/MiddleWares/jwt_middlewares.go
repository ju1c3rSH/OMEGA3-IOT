package MiddleWares

import (
	"OMEGA3-IOT/internal/service"
	"OMEGA3-IOT/internal/utils"
	"expvar"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	blacklistChecksTotal = expvar.NewInt("jwt_blacklist_checks_total")
	blacklistHitsTotal   = expvar.NewInt("jwt_blacklist_hits_total")
	blacklistErrorsTotal = expvar.NewInt("jwt_blacklist_errors_total")
)

type JWTAuth struct {
	blacklistService *service.TokenBlacklistService
}

func NewJWTAuth(blacklistService *service.TokenBlacklistService) *JWTAuth {
	return &JWTAuth{blacklistService: blacklistService}
}

func (j *JWTAuth) JwtAuthMiddleWare() gin.HandlerFunc {
	return func(context *gin.Context) {
		var tokenString string

		// Extract token from cookie or header
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

		// Parse and validate JWT
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

		// Check blacklist if service is available
		if j.blacklistService != nil {
			blacklistChecksTotal.Add(1)
			blacklisted, err := j.blacklistService.IsBlacklisted(
				context.Request.Context(), claims.JTI, claims.UUID, claims.IssuedAt,
			)
			if err != nil {
				blacklistErrorsTotal.Add(1)
				log.Printf("[JWTAuth] Redis error (fail-open): %v", err)
			}
			if blacklisted {
				blacklistHitsTotal.Add(1)
				context.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked"})
				context.Abort()
				return
			}
		}

		// Set context values
		context.Set("username", claims.UserName)
		context.Set("role", claims.Role)
		context.Set("user_uuid", claims.UUID)
		context.Set("ExpiresAt", claims.ExpiresAt)
		context.Set("jti", claims.JTI)
		context.Next()
	}
}