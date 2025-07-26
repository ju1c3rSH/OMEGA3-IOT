package handler

import (
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/handler/MiddleWares"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func RegRoutes(router *gin.Engine) {
	apiGroup := router.Group("/api/v1")
	apiGroup.GET("/GetTest", func(c *gin.Context) {
		msg := c.DefaultQuery("msg", "hello world")
		c.JSON(200, gin.H{
			"message": msg,
		})
	})

	apiGroup.POST("/Register", func(c *gin.Context) {
		var RU model.RegUser

		if err := c.ShouldBind(&RU); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			print("nothing in param")
			return
		}
		print(RU.Username)
		var existingUser model.User
		dbConn := db.DB
		if err := dbConn.Where("user_name = ?", RU.Username).First(&existingUser).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username has already taken"})
		}

		var hashedPassword, err = utils.HashPassword(RU.Password)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		newUser := model.User{
			UserName:     RU.Username,
			PasswordHash: hashedPassword,
			Type:         1,
			Status:       0,
			Role:         1,
			IP:           c.ClientIP(),
			CreatedAt:    time.Now().Unix(),
			UpdatedAt:    time.Now().Unix(),
			UserUUID:     utils.GenerateUUID().String(),
		}
		if err := dbConn.Create(&newUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User created successfully", "user": newUser})
	})

	apiGroup.POST("/Login", func(c *gin.Context) {
		var input model.LoginUser
		if err := c.ShouldBind(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user model.User
		dbConn := db.DB
		if err := dbConn.Where("user_name = ?", input.Username).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		if !utils.CheckPasswordHash(input.Password, user.PasswordHash) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		accessToken, err := utils.GenerateToken(user.UserName, user.UserUUID, user.Role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
			return
		}

		user.LastSeen = time.Now().Unix()
		user.IP = c.ClientIP()
		dbConn.Save(&user)
		c.JSON(http.StatusOK, gin.H{
			"message":      "Login successful",
			"access_token": accessToken,
			"user": gin.H{
				"id":       user.ID,
				"username": user.UserName,
				"role":     user.Role,
			},
		})
		//TODO refreshTiken

	})

	protected := apiGroup.Group("")
	protected.Use(MiddleWares.JwtAuthMiddleWare())
	{
		protected.GET("/GetUserInfo", func(c *gin.Context) {
			userID, exists := c.Get("user_id")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
				return
			}

			var user model.User
			if err := db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"user": gin.H{
					"id":         user.ID,
					"username":   user.UserName,
					"role":       user.Role,
					"created_at": user.CreatedAt,
					"last_seen":  user.LastSeen,
					"last_ip":    user.IP,
				},
			})
		})

		protected.POST("/AddDevice", func(c *gin.Context) {
			var input model.DeviceAddTemplate

			if err := c.ShouldBind(&input); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

		})
	}
}
