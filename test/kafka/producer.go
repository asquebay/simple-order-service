//go:build test

// добавил тег test выше, чтобы этот файл НЕ попал в сборку приложения
// этот код не зависит от приложения,
// и нужен только для тестирования отправки JSON-сообщений через кафку в бд
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/segmentio/kafka-go"
)

func main() {
	// конфигурация из config.yaml
	brokerAddress := "localhost:9092"
	topic := "orders"

	// JSON-сообщение
	message := `{
           "order_uid": "a112bbb3c3c44d5test",
           "track_number": "WBILMTESTTRACK",
           "entry": "WBIL",
           "delivery": { "name": "NixOS User", "phone": "+79001234567", "zip": "123456", "city": "NixOS City", "address": "Terminal street 1", "region": "Flake", "email": "nixos@example.com" },
           "payment": { "transaction": "a112bbb3c3c44d5test", "request_id": "", "currency": "RUB", "provider": "mir", "amount": 5000, "payment_dt": 1637907727, "bank": "sber", "delivery_cost": 500, "goods_total": 4500, "custom_fee": 0 },
           "items": [ { "chrt_id": 12345, "track_number": "WBILMTESTTRACK", "price": 4500, "rid": "some_random_id_123", "name": "NixOS T-Shirt", "sale": 0, "size": "M", "total_price": 4500, "nm_id": 67890, "brand": "Nix", "status": 202 } ],
           "locale": "en",
           "internal_signature": "",
           "customer_id": "nixos_user",
           "delivery_service": "cdek",
           "shardkey": "1",
           "sm_id": 1,
           "date_created": "2025-08-15T12:00:00Z",
           "oof_shard": "1"
        }`

	// настройки писателя (producer-а)
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokerAddress),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	log.Println("Sending message to Kafka...")
	err := writer.WriteMessages(context.Background(),
		kafka.Message{
			Value: []byte(message),
		},
	)
	if err != nil {
		log.Fatalf("Failed to write message: %v", err)
	}
	fmt.Println("Message sent successfully!")
}
