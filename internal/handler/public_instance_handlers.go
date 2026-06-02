package handler

import (
	"OMEGA3-IOT/internal/service"
	"OMEGA3-IOT/internal/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ==================== Public endpoints (no auth) ====================

// GetPublicInstancesHandlerFactory returns a handler that lists all public instances.
// Optional query param: ?type=<device_type>
func GetPublicInstancesHandlerFactory(publicInstanceService *service.PublicInstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceType := c.Query("type")

		// Try to get user UUID from JWT if present (non-mandatory)
		userUUID, _ := c.Get("user_uuid")
		var uid string
		if userUUID != nil {
			uid, _ = userUUID.(string)
		}

		instances, err := publicInstanceService.GetPublicInstances(deviceType, uid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to get public instances", err.Error()))
			return
		}

		c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{
			"count":     len(instances),
			"instances": instances,
		}, http.StatusOK, "Public instances retrieved successfully"))
	}
}

// GetPublicInstanceDetailHandlerFactory returns a handler that retrieves a single public instance.
func GetPublicInstanceDetailHandlerFactory(publicInstanceService *service.PublicInstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		instanceUUID := c.Param("instance_uuid")
		if instanceUUID == "" {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Missing instance_uuid"))
			return
		}

		userUUID, _ := c.Get("user_uuid")
		var uid string
		if userUUID != nil {
			uid, _ = userUUID.(string)
		}

		instance, err := publicInstanceService.GetPublicInstance(instanceUUID, uid)
		if err != nil {
			c.JSON(http.StatusNotFound, types.NewErrorResponse(http.StatusNotFound, "Public instance not found", err.Error()))
			return
		}

		c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(instance, http.StatusOK, "Public instance retrieved successfully"))
	}
}

// ==================== User endpoints (JWT required) ====================

// AddFavoriteHandlerFactory returns a handler that adds a public instance to user's favorites.
func AddFavoriteHandlerFactory(publicInstanceService *service.PublicInstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userUUID, exists := c.Get("user_uuid")
		if !exists {
			c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
			return
		}

		instanceUUID := c.Param("instance_uuid")
		if instanceUUID == "" {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Missing instance_uuid"))
			return
		}

		err := publicInstanceService.AddFavorite(userUUID.(string), instanceUUID)
		if err != nil {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Failed to add favorite", err.Error()))
			return
		}

		c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(nil, http.StatusOK, "Instance added to favorites"))
	}
}

// RemoveFavoriteHandlerFactory returns a handler that removes a public instance from user's favorites.
func RemoveFavoriteHandlerFactory(publicInstanceService *service.PublicInstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userUUID, exists := c.Get("user_uuid")
		if !exists {
			c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
			return
		}

		instanceUUID := c.Param("instance_uuid")
		if instanceUUID == "" {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Missing instance_uuid"))
			return
		}

		err := publicInstanceService.RemoveFavorite(userUUID.(string), instanceUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to remove favorite", err.Error()))
			return
		}

		c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(nil, http.StatusOK, "Instance removed from favorites"))
	}
}

// GetFavoritesHandlerFactory returns a handler that lists the user's favorited public instances.
func GetFavoritesHandlerFactory(publicInstanceService *service.PublicInstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userUUID, exists := c.Get("user_uuid")
		if !exists {
			c.JSON(http.StatusUnauthorized, types.NewErrorResponse(http.StatusUnauthorized, "User not authenticated"))
			return
		}

		instances, err := publicInstanceService.GetFavorites(userUUID.(string))
		if err != nil {
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to get favorites", err.Error()))
			return
		}

		c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{
			"count":     len(instances),
			"instances": instances,
		}, http.StatusOK, "Favorites retrieved successfully"))
	}
}

// ==================== Admin endpoints ====================

// TogglePublicHandlerFactory returns a handler that toggles the public status of an instance.
func TogglePublicHandlerFactory(publicInstanceService *service.PublicInstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		instanceUUID := c.Param("instance_uuid")
		if instanceUUID == "" {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Missing instance_uuid"))
			return
		}

		var input struct {
			IsPublic bool `json:"is_public"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(http.StatusBadRequest, "Invalid request body", err.Error()))
			return
		}

		err := publicInstanceService.TogglePublic(instanceUUID, input.IsPublic)
		if err != nil {
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse(http.StatusInternalServerError, "Failed to toggle public status", err.Error()))
			return
		}

		c.JSON(http.StatusOK, types.NewSuccessResponseWithCode(gin.H{
			"instance_uuid": instanceUUID,
			"is_public":     input.IsPublic,
		}, http.StatusOK, "Public status updated"))
	}
}