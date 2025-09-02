package main

import (
	"L0_project/internal/api"
	"L0_project/internal/cache"
	"L0_project/internal/config"
	"L0_project/internal/database"
	"L0_project/internal/kafka"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.Get()

	db, err := database.New(cfg.Postgres.URL)
	if err != nil {
		log.Fatalf("не удалось подключиться к postgres: %v", err)
	}
	defer db.Close()

	if err := db.ApplyMigrations("./internal/database/schema.sql"); err != nil {
		log.Fatalf("Не удалось применить migrations: %v", err)
	}

	orderCache := cache.NewLRUCache(cfg.Cache.Size)

	log.Println("Подготовка кеша...")
	orders, err := db.GetAllOrders(context.Background())
	if err != nil {
		log.Printf("не удалось получить все заказы для инициализации кеша: %v", err)
	} else {
		for _, order := range orders {
			orderCopy := order
			orderCache.Set(order.OrderUID, &orderCopy)
		}
		log.Printf("Кеш прогрет. Загружено %d заказов.", len(orders))
	}

	ctx, cancel := context.WithCancel(context.Background())
	consumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID, db, orderCache)
	go consumer.Start(ctx)

	handler := api.NewHandler(db, orderCache)
	router := api.NewRouter(handler)

	go func() {
		if err := api.StartServer(cfg.HTTP.Port, router); err != nil {
			log.Fatalf("не удалось запустить http сервер: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Завершение работы приложения...")
	cancel()
	log.Println("Приложение было успешно остановлено.")
}
