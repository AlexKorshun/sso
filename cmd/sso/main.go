package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlexKorshun/sso/internal/app"
	"github.com/AlexKorshun/sso/internal/config"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("starting application", slog.Any("config", cfg))

	application := app.New(log, cfg.GRPC.Port, cfg.DatabaseURL, cfg.TokenTTL)

	go application.GRPCServer.MustRun()
	// TODO: инициализировать логгер

	// TODO: инициализировать приложение (app)

	// TODO: запустить gRPC-сервер приложения

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	sign := <-stop
	log.Info("stopping application", slog.String("signal", sign.String()))
	application.GRPCServer.Stop()
	application.Storage.Close()
	log.Info("application stoppped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}
	return log

}
