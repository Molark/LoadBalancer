package main

import (
	"booking/internal/auth"
	"booking/internal/config"
	"booking/internal/handlers"
	"booking/internal/repository"
	"booking/internal/service"
	"log/slog"
	"net/http"
	"time"
)

func main() {

	cfg := config.Load()
	dsn := "postgres://" + cfg.DBUser + ":" + cfg.DBPassword + "@" + cfg.DBHost + ":" + cfg.DBPort + "/" + cfg.DBName + "?sslmode=disable"
	pool, err := repository.ConnectPool(dsn)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	repo := repository.NewRepository(pool)
	serviceS := service.NewService(repo)
	authS := auth.NewAuthService([]byte(cfg.JWTSecret))
	handler := handlers.NewHandler(serviceS, authS)
	mux := http.NewServeMux()
	handler.SetupRoutes(mux)
	server := http.Server{
		Addr:              ":8080",
		Handler:           handlers.CORSMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	err = server.ListenAndServe()
	if err != nil {
		slog.Error(err.Error())
	}
	slog.Info("server started at localhost:8080")
}
