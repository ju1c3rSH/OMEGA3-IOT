package handler

import (
	"captcha"
	"captcha/api"
	"captcha/theme/touhou"

	"github.com/gin-gonic/gin"
)

// InitCaptchaService creates and configures the EinkiCaptcha service.
func InitCaptchaService() *captcha.CaptchaService {
	cfg := captcha.DefaultConfig()
	cfg.Width = 300
	cfg.Height = 200
	cfg.Complexity = captcha.ComplexityMedium
	cfg.TTL = 300 // 5 minutes
	cfg.MaxObjects = 8
	cfg.MinObjects = 3
	cfg.QuestionTypes = []captcha.QuestionType{
		captcha.QuestionCounting,
		captcha.QuestionColor,
		captcha.QuestionCharacter,
		captcha.QuestionPosition,
	}

	service := captcha.New(
		captcha.WithConfig(cfg),
		captcha.WithServerSecret("einki-captcha-secret-change-me"),
	)

	// Register Touhou theme
	service.RegisterTheme(touhou.New())

	return service
}

// RegisterCaptchaRoutes registers captcha HTTP routes.
func RegisterCaptchaRoutes(v1 *gin.RouterGroup, service *captcha.CaptchaService) {
	handler := api.NewHandler(service)
	handler.RegisterRoutes(v1)
}
