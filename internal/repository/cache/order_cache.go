package cache

import (
	"sync"

	"github.com/asquebay/simple-order-service/internal/model"
)

// OrderCache — потокобезопасный in-memory кэш для заказов
type OrderCache struct {
	// sync.Map выбрал для обеспечения потокобезопасности
	// Ключ — string (OrderUID), значение — model.Order
	storage sync.Map
}

// NewOrderCache создаёт новый экземпляр кэша
func NewOrderCache() *OrderCache {
	return &OrderCache{}
}

// Set добавляет или обновляет заказ в кэше
func (c *OrderCache) Set(order model.Order) {
	c.storage.Store(order.OrderUID, order)
}

// Get извлекает заказ из кэша по его UID
// возвращает заказ и true, если он найден, иначе — пустую структуру и false
func (c *OrderCache) Get(orderUID string) (model.Order, bool) {
	value, ok := c.storage.Load(orderUID)
	if !ok {
		return model.Order{}, false
	}

	// выполняем безопасное приведение типа
	order, ok := value.(model.Order)
	return order, ok
}

// LoadAll загружает в кэш срез заказов
// используется для первоначального заполнения кэша при старте сервиса
func (c *OrderCache) LoadAll(orders []model.Order) {
	for _, order := range orders {
		c.Set(order)
	}
}
