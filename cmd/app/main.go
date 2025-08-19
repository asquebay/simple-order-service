package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asquebay/simple-order-service/internal/config"
	"github.com/asquebay/simple-order-service/internal/lib/logger"
	"github.com/asquebay/simple-order-service/internal/repository/cache"
	"github.com/asquebay/simple-order-service/internal/repository/postgres"
	"github.com/asquebay/simple-order-service/internal/service"
	httptransport "github.com/asquebay/simple-order-service/internal/transport/http"
	"github.com/asquebay/simple-order-service/internal/transport/kafka"
)

func main() {
	// 1. Инициализация конфигурации
	cfg := config.MustLoad("config/config.yaml")

	// 2. Инициализация логгера
	log := logger.New(cfg.Logger.Level)
	log.Info("starting simple-order-service", slog.String("log_level", cfg.Logger.Level))

	// 3. Инициализация репозитория (БД)
	initCtx := context.Background()
	dbpool, err := postgres.New(initCtx, cfg.Postgres)
	if err != nil {
		log.Error("failed to connect to postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbpool.Close()
	log.Info("successfully connected to postgres")

	orderRepo := postgres.NewOrderRepository(dbpool)

	// 4. Инициализация кэша
	orderCache := cache.NewOrderCache()
	log.Info("order cache initialized")

	// 5. Инициализация сервисного слоя
	orderSvc := service.NewOrderService(orderRepo, orderCache, log)

	// 6. Восстановление кэша из БД при старте
	err = orderSvc.RestoreCache(context.Background())
	if err != nil {
		// не фатальная ошибка, сервис может работать и с пустым кэшем
		log.Error("failed to restore cache", slog.String("error", err.Error()))
	}

	// 7. Инициализация и запуск Kafka-консьюмера
	consumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID, orderSvc, log)
	ctx, cancel := context.WithCancel(context.Background())
	go consumer.Run(ctx)

	// 8. Инициализация и запуск HTTP-сервера
	handler := httptransport.NewHandler(orderSvc, log)
	httpServer := httptransport.NewServer(cfg.HTTPServer.Port, handler, cfg.HTTPServer.Timeout)
	log.Info("starting http server", slog.String("port", cfg.HTTPServer.Port))

	go func() {
		if err := httpServer.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed to start", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// 9. Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Info("shutting down application")
	cancel() // сигнал для консьюмера на завершение

	// создаем контекст с таймаутом для шатдауна сервера
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("http server shutdown failed", slog.String("error", err.Error()))
	}

	if err := consumer.Close(); err != nil {
		log.Error("error closing kafka consumer", slog.String("error", err.Error()))
	}

	log.Info("application stopped")
}
