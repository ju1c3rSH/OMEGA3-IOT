package http_api

import (
	"OMEGA3-IOT/internal/config"
	"OMEGA3-IOT/internal/handler"
	"OMEGA3-IOT/internal/handler/MiddleWares"
	"OMEGA3-IOT/internal/logger"
	"OMEGA3-IOT/internal/push"
	"OMEGA3-IOT/internal/service"
	"OMEGA3-IOT/internal/utils"
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
)

// @title IOT HTTP API
// @version 0.1
// @description IOT device management API
// @host localhost:1222
// @BasePath /api/v1

func Run(mqttService *service.MQTTService, userHandler *handler.UserHandler, deviceHandler *handler.DeviceHandler, logHandler *logger.LogHandler, config config.Config, deviceService *service.DeviceService, deviceShareService *service.DeviceShareService, deviceFolderHandler *handler.DeviceFolderHandler, jwtAuth *MiddleWares.JWTAuth, pushHandler *push.PushHandler, userGroupHandler *handler.UserGroupHandler, adminHandler *handler.AdminHandler, publicInstanceService *service.PublicInstanceService, shutdown <-chan struct{}) error {

	log.Println("[HTTP_API] Run function called")

	if config.Server.Debug {
		gin.SetMode(gin.DebugMode)
		// pprof 仅 debug 暴露于本地回环，避免生产泄漏
		go func() {
			log.Println("[HTTP_API] pprof enabled on 127.0.0.1:6060")
			if err := http.ListenAndServe("127.0.0.1:6060", nil); err != nil {
				log.Printf("[HTTP_API] pprof server error: %v", err)
			}
		}()
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	if err := r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.1/32"}); err != nil {
		log.Printf("[WARN] failed to set trusted proxies: %v", err)
	}
	r.ForwardedByClientIP = true
	r.Use(gin.Recovery())
	// 异步采样日志：health 按 IP 1/min，其它全量，缓冲 4096 满丢弃
	sampledLogger := MiddleWares.NewSampledLogger()
	r.Use(sampledLogger.Middleware())

	// 单一全局 CORS（已移除 http_api_routes.go 自定义 Cors，避免双头/双 abort）
	r.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"}, // 开发环境可用，生产环境替换为具体域名
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        12 * time.Hour,
	}))

	handler.RegRoutes(r, userHandler, deviceHandler, logHandler, deviceService, deviceShareService, deviceFolderHandler, mqttService, jwtAuth, pushHandler, userGroupHandler, adminHandler, publicInstanceService)

	log.Println("Starting server on :" + config.Server.Port)

	srv := &http.Server{
		Addr:              ":" + config.Server.Port,
		Handler:           r,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	var certFile, keyFile string
	errCh := make(chan error, 1)
	serve := func() {
		if config.Server.TLSEnabled {
			errCh <- srv.ListenAndServeTLS(certFile, keyFile)
			return
		}
		errCh <- srv.ListenAndServe()
	}

	if config.Server.TLSEnabled {
		var err error
		certFile, keyFile, err = utils.EnsureCertificates(config.Server.CertFile, config.Server.KeyFile)
		if err != nil {
			log.Fatalf("Failed to prepare TLS certificates: %v", err)
		}
		log.Printf("TLS enabled: cert=%s, key=%s", certFile, keyFile)
		srv.TLSConfig = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
			CurvePreferences: []tls.CurveID{
				tls.X25519,
				tls.CurveP256,
			},
		}
	}
	go serve()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-shutdown:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("[HTTP_API] graceful shutdown error: %v", err)
		}
		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		log.Println("[HTTP_API] server shut down gracefully")
		return nil
	}
}
