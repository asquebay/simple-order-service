package service

import (
	"context"

	"github.com/asquebay/simple-order-service/internal/model"
)

// OrderRepository определяет контракт для хранилища заказов в БД
type OrderRepository interface {
	CreateOrder(ctx context.Context, order model.Order) error
	GetAllOrders(ctx context.Context) ([]model.Order, error)
	GetOrderByUID(ctx context.Context, uid string) (model.Order, error)
}

// OrderCache определяет контракт для in-memory кэша заказов
type OrderCache interface {
	Set(order model.Order)
	Get(orderUID string) (model.Order, bool)
	LoadAll(orders []model.Order)
}
