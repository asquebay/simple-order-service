package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"

	"github.com/asquebay/simple-order-service/internal/model"

	"github.com/segmentio/kafka-go"
)

// OrderCreator — это интерфейс, который абстрагирует консьюмер
// от конкретной реализации сервисного слоя
type OrderCreator interface {
	CreateOrder(ctx context.Context, order model.Order) error
}

// Consumer представляет собой консьюмер сообщений Kafka
type Consumer struct {
	reader  *kafka.Reader
	service OrderCreator
	log     *slog.Logger
}

// NewConsumer создает новый экземпляр консьюмера
func NewConsumer(brokers []string, topic, groupID string, service OrderCreator, log *slog.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		GroupID: groupID,
		Topic:   topic,
		// StartOffset: kafka.FirstOffset, // читаем с начала, если группа новая или смещения удалены
		// я добавил эту строку, т.к. были многочисленные ошибки с кафкой при развёртывании в докер-композе,
		// пока что я отбросил развёртывание сервиса в контейнерах, но в будущем всё-таки разверну,
		// поэтому строка может оказаться мне нужной, а пока пусть будет закомментирована
	})

	return &Consumer{
		reader:  reader,
		service: service,
		log:     log,
	}
}

// Run запускает цикл чтения сообщений из Kafka
// эта функция блокирующая, поэтому она запускается в отдельной горутине
func (c *Consumer) Run(ctx context.Context) {
	log := c.log.With(slog.String("component", "kafka_consumer"))
	log.Info("Kafka consumer started")

	for {
		// проверка на отмену контекста
		select {
		case <-ctx.Done():
			log.Info("Context cancelled, stopping consumer.")
			return
		default:
			// FetchMessage блокирует до тех пор, пока не придет новое сообщение или не возникнет ошибка
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				// если контекст был отменен во время ожидания, это нормальное завершение
				if errors.Is(err, context.Canceled) {
					return
				}
				// если ридер был закрыт, тоже выходим
				if errors.Is(err, io.EOF) {
					log.Info("Kafka reader closed")
					return
				}
				log.Error("failed to fetch message", slog.String("error", err.Error()))
				continue // пробуем снова
			}

			log.Info("received message", slog.String("topic", msg.Topic), slog.Int("partition", msg.Partition), slog.Int64("offset", msg.Offset))

			// 1. Пытаемся обработать
			if err := c.handleMessage(ctx, msg); err != nil {
				log.Error("failed to handle message", slog.String("error", err.Error()))
				// сообщение НЕ подтверждаем — пусть Kafka отдаст его снова
				continue
			}

			// подтверждаем получение сообщения, чтобы Kafka не отправила его снова
			// это ВАЖНО сделать ПОСЛЕ успешной обработки
			// 2. Всё прошло — фиксируем offset
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				log.Error("failed to commit message", slog.String("error", err.Error()))
			}
		}
	}
}

// handleMessage парсит и обрабатывает одно сообщение
func (c *Consumer) handleMessage(ctx context.Context, msg kafka.Message) error {
	var order model.Order

	// распарсим JSON
	if err := json.Unmarshal(msg.Value, &order); err != nil {
		// сообщение невалидно. Логируем и игнорируем, согласно условии задачи
		c.log.Warn("failed to unmarshal message, skipping", slog.String("error", err.Error()))
		return nil // возвращаем nil, так как перечитывать это сообщение бессмысленно
	}

	// валидация данных
	if err := order.Validate(); err != nil {
		// данные не прошли валидацию (например, отсутствуют обязательные поля)
		// логируем и игнорируем
		c.log.Warn("message validation failed, skipping",
			slog.String("error", err.Error()),
			slog.String("order_uid", order.OrderUID),
		)
		return nil // также не перечитываем
	}

	// передаём заказ в сервисный слой для сохранения в БД и кэше
	if err := c.service.CreateOrder(ctx, order); err != nil {
		// если произошла ошибка при сохранении (например, дубликат),
		// логируем её и решаем, нужно ли повторять попытку
		// в данном случае, если это ошибка дубликата, повторять не нужно
		c.log.Error("failed to create order in service",
			slog.String("error", err.Error()),
			slog.String("order_uid", order.OrderUID),
		)
		return err // возвращаем ошибку, чтобы вызывающая функция могла ее обработать
	}

	c.log.Info("order successfully processed", slog.String("order_uid", order.OrderUID))
	return nil
}

// gracefull shutdown консьюмера
func (c *Consumer) Close() error {
	c.log.Info("Closing kafka consumer")
	return c.reader.Close()
}
