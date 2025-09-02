package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type Order struct {
	OrderUID    string      `json:"order_uid"`
	TrackNumber string      `json:"track_number"`
	Entry       string      `json:"entry"`
	Delivery    interface{} `json:"delivery"`
	Payment     interface{} `json:"payment"`
	Items       interface{} `json:"items"`
	Locale      string      `json:"locale"`
	CustomerID  string      `json:"customer_id"`
	DateCreated time.Time   `json:"date_created"`
}

func main() {
	topic := "orders"
	brokerAddress := "localhost:9092"

	w := &kafka.Writer{
		Addr:     kafka.TCP(brokerAddress),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer w.Close()

	jsonFile, err := os.Open("./model.json")
	if err != nil {
		log.Fatalf("не удалось открыть model.json: %v", err)
	}
	defer jsonFile.Close()

	byteValue, _ := os.ReadFile("./model.json")
	var baseOrder map[string]interface{}
	json.Unmarshal(byteValue, &baseOrder)

	fmt.Println("Продюсер запущен. Нажмите CTRL+C для остановки.")
	for {
		newOrder := make(map[string]interface{})
		for k, v := range baseOrder {
			newOrder[k] = v
		}

		orderUID := uuid.New().String()
		trackNumber := fmt.Sprintf("WBILM%d", 1000000+rand.Intn(9000000))

		newOrder["order_uid"] = orderUID
		newOrder["track_number"] = trackNumber
		newOrder["date_created"] = time.Now()

		payment := newOrder["payment"].(map[string]interface{})
		payment["transaction"] = orderUID
		items := newOrder["items"].([]interface{})
		for _, item := range items {
			itemMap := item.(map[string]interface{})
			itemMap["track_number"] = trackNumber
		}

		orderBytes, err := json.Marshal(newOrder)
		if err != nil {
			log.Printf("не удалось преобразовать заказ в JSON: %v", err)
			continue
		}

		err = w.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte(orderUID),
				Value: orderBytes,
			},
		)
		if err != nil {
			log.Fatalf("не удалось отправить сообщение: %v", err)
		}

		fmt.Printf("Отправлен заказ с UID: %s\n", orderUID)
		time.Sleep(2 * time.Second)
	}
}
