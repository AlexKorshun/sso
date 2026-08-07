package app

import (
	"log/slog"
	"time"

	grpcapp "github.com/AlexKorshun/sso/internal/app/grpc"
)

type App struct {
	GRPCServer *grpcapp.App
}

func New(
	log *slog.Logger,
	grpcPort int,
	storagePath string,
	tokenTTL time.Duration) *App {

	//TODO: инициализировать storage

	//TODO: инициализировать сервисный слой

	grpcApp := grpcapp.New(log, grpcPort)

	return &App{GRPCServer: grpcApp}
}
