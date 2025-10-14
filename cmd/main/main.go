package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"L0_project/internal/api"
	"L0_project/internal/cache"
	"L0_project/internal/config"
	"L0_project/internal/database"
	"L0_project/internal/kafka"
)

func main() {
	cfg := config.Get()

	db, err := database.New(cfg.Postgres.URL)
	if err != nil {
		log.Fatalf("не удалось подключиться к postgres: %v", err)
	}
	defer db.Close()

	// Миграции теперь выполняются вне кода (см. migrations/).

	orderCache := cache.NewLRUCache(cfg.Cache.Size)

	log.Println("Подготовка кеша...")
	orders, err := db.GetAllOrders(context.Background())
	if err != nil {
		log.Printf("не удалось получить все заказы для инициализации кеша: %v", err)
	} else {
		for _, order := range orders {
			orderCopy := order
			orderCache.Add(order.OrderUID, &orderCopy)
		}
		log.Printf("Кеш прогрет. Загружено %d заказов.", len(orders))
	}

	ctx, cancel := context.WithCancel(context.Background())
	consumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID, db, orderCache)
	go consumer.Start(ctx)

	handler := api.NewHandler(db, orderCache)
	router := api.NewRouter(handler)

	srv := api.NewServer(cfg.HTTP.Port, router)

	// Запускаем HTTP сервер в отдельной горутине
	
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.ListenAndServe()
	}()

	// Ожидаем сигнал завершения или ошибку сервера
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Println("Получен сигнал завершения")
	case err := <-serverErr:
		if err != nil {
			log.Printf("HTTP сервер завершился с ошибкой: %v", err)
		}
	}

	log.Println("Завершение работы приложения...")

	// Сигнал для остановки consumer
	
	cancel()
	
	// Плавно останавливаем HTTP сервер
	
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Ошибка при остановке сервера: %v", err)
	}

	log.Println("Приложение было успешно остановлено.")
}
