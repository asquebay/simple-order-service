package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/asquebay/simple-order-service/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// New создает и возвращает новый пул соединений с PostgreSQL
func New(ctx context.Context, cfg config.Postgres) (*pgxpool.Pool, error) {
	const op = "repository.postgres.postgres.New"

	dsn := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to parse pgx config: %w", op, err)
	}

	// настройка пула соединений
	poolConfig.MaxConns = 10
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	dbpool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create connection pool: %w", op, err)
	}

	// проверяем, что соединение установлено
	if err := dbpool.Ping(ctx); err != nil {
		dbpool.Close()
		return nil, fmt.Errorf("%s: failed to ping database: %w", op, err)
	}

	return dbpool, nil
}
