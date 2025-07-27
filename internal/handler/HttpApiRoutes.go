package handler

import (
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/handler/MiddleWares"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strconv"
	"strings"
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username has already taken,err: " + err.Error()})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user,err: " + err.Error()})
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

		dbConn.Model(&user).Updates(map[string]interface{}{
			"last_seen": time.Now().Unix(),
			"ip":        c.ClientIP(),
		})
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
			var input struct {
				Name        string `json:"name" binding:"required"`
				DeviceType  int    `json:"device_type" binding:"required"`
				Description string `json:"description,omitempty"`
			}

			if err := c.ShouldBind(&input); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			userUUID, exists := c.Get("user_uuid")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
				return
			}

			// 验证设备类型
			deviceType, valid := model.GlobalDeviceTypeManager.GetById(input.DeviceType)
			if !valid {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported device type: " + strconv.Itoa(input.DeviceType)})
				return
			}

			// 生成设备实例
			//deviceUUID := utils.GenerateUUID().String()
			instance, _ := model.NewInstanceFromConfig(input.Name, userUUID.(string), deviceType)
			instance.Description = input.Description

			// 保存到数据库 - 现在会正确保存到 instances 表
			dbConn := db.DB.Session(&gorm.Session{SkipDefaultTransaction: true})
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := dbConn.WithContext(ctx).Create(&instance).Error; err != nil {
				// 检查具体错误类型
				if strings.Contains(err.Error(), "Duplicate entry") {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Device name already exists"})
					return
				}
				log.Printf("Failed to create device: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create device: " + err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "Device created successfully",
				"device": gin.H{
					"id":          instance.ID,
					"uuid":        instance.InstanceUUID,
					"name":        instance.Name,
					"type":        instance.Type,
					"online":      instance.Online,
					"description": instance.Description,
					"created_at":  instance.AddTime,
					"last_seen":   instance.LastSeen,
					"properties":  instance.Properties.Items,
				},
			})
		})
	}
}
