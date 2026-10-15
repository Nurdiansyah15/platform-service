// Command worker mengirim event outbox ke endpoint webhook (spec §2.9):
// fan-out events → webhook_deliveries, lalu kirim delivery yang jatuh tempo
// dengan retry exponential backoff. Aman dijalankan lebih dari satu replika
// (klaim delivery memakai FOR UPDATE SKIP LOCKED + lease).
package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.Database)
	if err != nil {
		appLogger.Error("gagal koneksi database", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	dispatcher, err := bootstrap.NewDispatcher(cfg, db, appLogger, nil)
	if err != nil {
		appLogger.Error("gagal bootstrap worker", slog.Any("error", err))
		os.Exit(1)
	}

	appLogger.Info("webhook worker berjalan",
		slog.Duration("poll_interval", cfg.Webhook.PollInterval),
		slog.Int("batch_size", cfg.Webhook.BatchSize))

	ticker := time.NewTicker(cfg.Webhook.PollInterval)
	defer ticker.Stop()
	for {
		// Siklus berjalan dengan context terpisah agar pengiriman yang sedang
		// berlangsung selesai (dan hasilnya tersimpan) saat shutdown.
		runCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout+cfg.Webhook.Timeout)
		created, sent, err := dispatcher.RunOnce(runCtx)
		cancel()
		switch {
		case err != nil:
			appLogger.Error("siklus webhook gagal", slog.Any("error", err))
		case created > 0 || sent > 0:
			appLogger.Info("siklus webhook", slog.Int("deliveries_created", created), slog.Int("deliveries_sent", sent))
		}

		select {
		case <-ctx.Done():
			appLogger.Info("menghentikan worker...")
			return
		case <-ticker.C:
		}
	}
}
