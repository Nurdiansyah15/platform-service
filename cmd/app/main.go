// Command app menjalankan HTTP API Central SaaS Platform.
//
//	@title			Central SaaS Platform API
//	@version		1.0
//	@description	Identity, tenant, catalog, subscription, entitlement, authorization, dan licensing untuk solution.
//	@BasePath		/
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"platform-service/internal/bootstrap"
	"platform-service/internal/config"
	"platform-service/internal/infrastructure/database"
	"platform-service/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("gagal load config: %v", err)
	}

	appLogger := logger.New(cfg.App.Env, cfg.App.LogFormat)
	slog.SetDefault(appLogger)

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.Database)
	if err != nil {
		appLogger.Error("gagal koneksi database", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	app, err := bootstrap.New(ctx, bootstrap.Options{Config: cfg, DB: db, Logger: appLogger})
	if err != nil {
		appLogger.Error("gagal bootstrap aplikasi", slog.Any("error", err))
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.App.Port,
		Handler:           app.Engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		appLogger.Info("HTTP server berjalan",
			slog.String("addr", srv.Addr),
			slog.String("deployment_mode", string(cfg.App.DeploymentMode)),
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLogger.Error("HTTP server berhenti", slog.Any("error", err))
			stop()
		}
	}()

	<-ctx.Done()
	appLogger.Info("menghentikan server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("graceful shutdown gagal", slog.Any("error", err))
	}
}
