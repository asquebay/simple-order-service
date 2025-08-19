package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/asquebay/simple-order-service/internal/model"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrderRepository инкапсулирует логику работы с заказами в БД
type OrderRepository struct {
	db *pgxpool.Pool
	sq squirrel.StatementBuilderType
}

// NewOrderRepository создает новый экземпляр репозитория
func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{
		db: db,
		// использую плейсхолдеры в стиле PostgreSQL ($1, $2, $3,...)
		sq: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// CreateOrder сохраняет полный заказ в базу данных в рамках одной транзакции
func (r *OrderRepository) CreateOrder(ctx context.Context, order model.Order) error {
	const op = "repository.postgres.order.CreateOrder"

	// начинаем транзакцию
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	// гарантируем откат транзакции в случае любой ошибки
	defer tx.Rollback(ctx)

	// 1. Вставка в таблицу orders
	sql, args, err := r.sq.Insert("orders").
		Columns(
			"order_uid", "track_number", "entry", "locale", "internal_signature",
			"customer_id", "delivery_service", "shardkey", "sm_id", "date_created", "oof_shard",
		).
		Values(
			order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
			order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: failed to build orders insert query: %w", op, err)
	}
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("%s: failed to insert into orders: %w", op, err)
	}

	// 2. Вставка в таблицу deliveries
	sql, args, err = r.sq.Insert("deliveries").
		Columns("order_uid", "name", "phone", "zip", "city", "address", "region", "email").
		Values(
			order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
			order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: failed to build deliveries insert query: %w", op, err)
	}
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("%s: failed to insert into deliveries: %w", op, err)
	}

	// 3. Вставка в таблицу payments
	sql, args, err = r.sq.Insert("payments").
		Columns(
			"transaction_uid", "request_id", "currency", "provider", "amount",
			"payment_dt", "bank", "delivery_cost", "goods_total", "custom_fee",
		).
		Values(
			order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency, order.Payment.Provider,
			order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank, order.Payment.DeliveryCost,
			order.Payment.GoodsTotal, order.Payment.CustomFee,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: failed to build payments insert query: %w", op, err)
	}
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("%s: failed to insert into payments: %w", op, err)
	}

	// 4. Вставка в таблицу items (в цикле)
	for _, item := range order.Items {
		sql, args, err = r.sq.Insert("items").
			Columns(
				"order_uid", "chrt_id", "track_number", "price", "rid", "name",
				"sale", "size", "total_price", "nm_id", "brand", "status",
			).
			Values(
				order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name,
				item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status,
			).
			ToSql()
		if err != nil {
			return fmt.Errorf("%s: failed to build items insert query for chrt_id %d: %w", op, item.ChrtID, err)
		}
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			return fmt.Errorf("%s: failed to insert item with chrt_id %d: %w", op, item.ChrtID, err)
		}
	}

	// если все прошло успешно, подтверждаем транзакцию
	return tx.Commit(ctx)
}

// GetAllOrders извлекает все заказы из базы данных
// этот метод может быть ресурсоёмким на больших объемах данных
// он предназначен для восстановления кэша при старте
func (r *OrderRepository) GetAllOrders(ctx context.Context) ([]model.Order, error) {
	const op = "repository.postgres.order.GetAllOrders"

	// 1. Получаем все основные данные заказов
	query := `
		SELECT
			o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature, o.customer_id,
			o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
			d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
			p.transaction_uid, p.request_id, p.currency, p.provider, p.amount, p.payment_dt,
			p.bank, p.delivery_cost, p.goods_total, p.custom_fee
		FROM orders o
		JOIN deliveries d ON o.order_uid = d.order_uid
		JOIN payments p ON o.order_uid = p.transaction_uid
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query orders: %w", op, err)
	}
	defer rows.Close()

	ordersMap := make(map[string]*model.Order)
	orderUIDs := []string{}

	for rows.Next() {
		var o model.Order
		err := rows.Scan(
			&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature, &o.CustomerID,
			&o.DeliveryService, &o.Shardkey, &o.SmID, &o.DateCreated, &o.OofShard,
			&o.Delivery.Name, &o.Delivery.Phone, &o.Delivery.Zip, &o.Delivery.City, &o.Delivery.Address, &o.Delivery.Region, &o.Delivery.Email,
			&o.Payment.Transaction, &o.Payment.RequestID, &o.Payment.Currency, &o.Payment.Provider, &o.Payment.Amount, &o.Payment.PaymentDt,
			&o.Payment.Bank, &o.Payment.DeliveryCost, &o.Payment.GoodsTotal, &o.Payment.CustomFee,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan order row: %w", op, err)
		}
		ordersMap[o.OrderUID] = &o
		orderUIDs = append(orderUIDs, o.OrderUID)
	}

	if len(orderUIDs) == 0 {
		return []model.Order{}, nil // нет заказов — возвращаем пустой слайс
	}

	// 2. Получаем все товары для найденных заказов
	itemsQuery := `
		SELECT order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
		FROM items
		WHERE order_uid = ANY($1)
	`
	itemRows, err := r.db.Query(ctx, itemsQuery, orderUIDs)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query items: %w", op, err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item model.Item
		var orderUID string
		err := itemRows.Scan(
			&orderUID, &item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name,
			&item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan item row: %w", op, err)
		}

		if order, ok := ordersMap[orderUID]; ok {
			order.Items = append(order.Items, item)
		}
	}

	// 3. Конвертируем map в слайс
	result := make([]model.Order, 0, len(ordersMap))
	for _, order := range ordersMap {
		result = append(result, *order)
	}

	return result, nil
}

var ErrOrderNotFound = errors.New("order not found")

// GetOrderByUID извлекает один заказ из базы данных по его UID
func (r *OrderRepository) GetOrderByUID(ctx context.Context, uid string) (model.Order, error) {
	const op = "repository.postgres.order.GetOrderByUID"

	// 1. Получаем основные данные заказа одним запросом
	query := `
		SELECT
			o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature, o.customer_id,
			o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
			d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
			p.transaction_uid, p.request_id, p.currency, p.provider, p.amount, p.payment_dt,
			p.bank, p.delivery_cost, p.goods_total, p.custom_fee
		FROM orders o
		JOIN deliveries d ON o.order_uid = d.order_uid
		JOIN payments p ON o.order_uid = p.transaction_uid
		WHERE o.order_uid = $1
	`
	var order model.Order
	err := r.db.QueryRow(ctx, query, uid).Scan(
		&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature, &order.CustomerID,
		&order.DeliveryService, &order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard,
		&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip, &order.Delivery.City, &order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email,
		&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency, &order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDt,
		&order.Payment.Bank, &order.Payment.DeliveryCost, &order.Payment.GoodsTotal, &order.Payment.CustomFee,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Order{}, fmt.Errorf("%s: %w", op, ErrOrderNotFound)
		}
		return model.Order{}, fmt.Errorf("%s: failed to query order: %w", op, err)
	}

	// 2. Получаем все товары для этого заказа
	itemsQuery := `
		SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
		FROM items
		WHERE order_uid = $1
	`
	rows, err := r.db.Query(ctx, itemsQuery, uid)
	if err != nil {
		return model.Order{}, fmt.Errorf("%s: failed to query items: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.Item
		err := rows.Scan(
			&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name,
			&item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status,
		)
		if err != nil {
			return model.Order{}, fmt.Errorf("%s: failed to scan item row: %w", op, err)
		}
		order.Items = append(order.Items, item)
	}

	return order, nil
}
