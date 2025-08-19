package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/asquebay/simple-order-service/internal/model"
	"github.com/asquebay/simple-order-service/internal/repository/postgres"
)

// OrderService инкапсулирует бизнес-логику работы с заказами
type OrderService struct {
	repo  OrderRepository
	cache OrderCache
	log   *slog.Logger
}

// NewOrderService создаёт новый экземпляр сервиса заказов
// он принимает интерфейсы, а не конкретные типы, для гибкости и тестируемости
func NewOrderService(repo OrderRepository, cache OrderCache, log *slog.Logger) *OrderService {
	return &OrderService{
		repo:  repo,
		cache: cache,
		log:   log,
	}
}

// CreateOrder обрабатывает создание нового заказа
// сначала он сохраняет заказ в постоянное хранилище (БД),
// и только в случае успеха добавляет его в кэш
func (s *OrderService) CreateOrder(ctx context.Context, order model.Order) error {
	const op = "service.OrderService.CreateOrder"
	log := s.log.With(slog.String("op", op), slog.String("order_uid", order.OrderUID))

	log.Info("attempting to create order")

	// 1. Сохраняем в БД. Это основной источник правды
	err := s.repo.CreateOrder(ctx, order)
	if err != nil {
		log.Error("failed to save order to repository", slog.String("error", err.Error()))
		// ошибку не маскируем, а оборачиваем для контекста
		return fmt.Errorf("%s: %w", op, err)
	}

	// 2. Если в БД сохранилось успешно, обновляем кэш
	s.cache.Set(order)
	log.Info("order created and cached successfully")

	return nil
}

// GetOrderByUID получает заказ по его ID
// сначала ищет в кэше, и только если там нет — обращается к БД
func (s *OrderService) GetOrderByUID(ctx context.Context, uid string) (model.Order, error) {
	const op = "service.OrderService.GetOrderByUID"
	log := s.log.With(slog.String("op", op), slog.String("order_uid", uid))

	// 1. Пытаемся получить из кэша для максимальной скорости
	order, found := s.cache.Get(uid)
	if found {
		log.Debug("order found in cache")
		return order, nil
	}

	log.Debug("order not found in cache, will check repository")

	// 2. Если в кэше нет, идем в БД
	order, err := s.repo.GetOrderByUID(ctx, uid)
	if err != nil {
		// не логируем как ошибку, если просто не найдено
		if !errors.Is(err, postgres.ErrOrderNotFound) {
			log.Error("failed to get order from repository", slog.String("error", err.Error()))
		}
		return model.Order{}, fmt.Errorf("%s: %w", op, err)
	}

	// 3. Раз уж мы достали заказ из БД, стоит положить его в кэш
	s.cache.Set(order)
	log.Info("order found in repository and now cached")

	return order, nil
}

// RestoreCache восстанавливает состояние кэша из базы данных при старте
func (s *OrderService) RestoreCache(ctx context.Context) error {
	const op = "service.OrderService.RestoreCache"
	log := s.log.With(slog.String("op", op))

	log.Info("starting cache restoration from database")

	orders, err := s.repo.GetAllOrders(ctx)
	if err != nil {
		log.Error("failed to get all orders from repository", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	s.cache.LoadAll(orders)

	log.Info("cache restored successfully", slog.Int("orders_count", len(orders)))
	return nil
}
