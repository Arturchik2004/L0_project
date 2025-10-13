package kafka

import (
	"L0_project/internal/cache"
	"L0_project/internal/database"
	"L0_project/internal/model"
	"context"
	"encoding/json"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader   *kafka.Reader
	db       database.OrderStorage
	cache    cache.OrderCache
	validate *validator.Validate
}

func NewConsumer(brokers []string, topic, groupID string, db database.OrderStorage, cache cache.OrderCache) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	return &Consumer{reader: r, db: db, cache: cache, validate: validator.New()}
}

func (c *Consumer) Start(ctx context.Context) {
	log.Println("Kafka запущен...")
	for {
		select {
		case <-ctx.Done():
			log.Println("Завершение работы Kafka...")
			c.reader.Close()
			return
		default:
			m, err := c.reader.FetchMessage(ctx)
			if err != nil {
				log.Printf("не удалось получить сообщение: %v", err)
				continue
			}

			var order model.Order
			if err := json.Unmarshal(m.Value, &order); err != nil {
				log.Printf("не удалось разобрать сообщение: %v. Сообщение: %s", err, string(m.Value))
				c.reader.CommitMessages(ctx, m)
				continue
			}

			if c.validate != nil {
				if err := c.validate.Struct(&order); err != nil {
					log.Printf("невалидные данные в заказе %s: %v", order.OrderUID, err)
					c.reader.CommitMessages(ctx, m)
					continue
				}
			}

			if err := c.db.SaveOrder(ctx, &order); err != nil {
				log.Printf("не удалось сохранить заказ %s в базу данных: %v", order.OrderUID, err)
				continue
			}

			log.Printf("Заказ %s успешно сохранен в базу данных", order.OrderUID)
			c.cache.Add(order.OrderUID, &order)
			log.Printf("Заказ %s успешно закэширован", order.OrderUID)

			if err := c.reader.CommitMessages(ctx, m); err != nil {
				log.Printf("не удалось подтвердить сообщение: %v", err)
			}
		}
	}
}
