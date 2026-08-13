package app

import (
	"context"
	"log/slog"
	"time"

	grpcapp "github.com/AlexKorshun/sso/internal/app/grpc"
	"github.com/AlexKorshun/sso/internal/services/auth"
	"github.com/AlexKorshun/sso/internal/storage/postgres"
)

type App struct {
	GRPCServer *grpcapp.App
	Storage    *postgres.Storage
}

func New(
	log *slog.Logger,
	grpcPort int,
	databaseURL string,
	tokenTTL time.Duration) *App {

	storage, err := postgres.New(context.Background(), databaseURL)
	if err != nil {
		panic(err)
	}

	authService := auth.New(log, storage, storage, storage, tokenTTL)
	grpcApp := grpcapp.New(log, authService, grpcPort)

	return &App{GRPCServer: grpcApp, Storage: storage}
}
