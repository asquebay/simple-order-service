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
           "order_uid": "my-first-test-order-01",
           "track_number": "WBILMTESTTRACK",
           "entry": "WBIL",
           "delivery": { "name": "Ivan Ivanov", "phone": "+9720000000", "zip": "2639809", "city": "Moscow", "address": "Mira street 15", "region": "Moscow Region", "email": "test@gmail.com" },
           "payment": { "transaction": "my-first-test-order-01", "request_id": "", "currency": "USD", "provider": "wbpay", "amount": 1817, "payment_dt": 1637907727, "bank": "alpha", "delivery_cost": 1500, "goods_total": 317, "custom_fee": 0 },
           "items": [ { "chrt_id": 9934930, "track_number": "WBILMTESTTRACK", "price": 453, "rid": "ab4219087a764ae0btest", "name": "Mascaras", "sale": 30, "size": "0", "total_price": 317, "nm_id": 2389222, "brand": "Vivienne Sabo", "status": 202 } ],
           "locale": "en",
           "internal_signature": "",
           "customer_id": "test",
           "delivery_service": "meest",
           "shardkey": "9",
           "sm_id": 99,
           "date_created": "2021-11-26T06:22:19Z",
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
