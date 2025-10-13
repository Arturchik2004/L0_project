package config

import (
	"log"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HTTP struct {
		Port string `env:"HTTP_PORT" env-default:"8081"`
	}
	Postgres struct {
		URL string `env:"POSTGRES_URL" env-default:"postgres://postgres:123@localhost:5432/GoLangWB?sslmode=disable"`
	}
	Kafka struct {
		Brokers []string `env:"KAFKA_BROKERS" env-default:"localhost:9092"`
		Topic   string   `env:"KAFKA_TOPIC" env-default:"orders"`
		GroupID string   `env:"KAFKA_GROUP_ID" env-default:"orders-group"`
	}
	Cache struct {
		Size int `env:"CACHE_SIZE" env-default:"100"`
	}
}

var (
	cfg  Config
	once sync.Once
)

func Get() *Config {
	once.Do(func() {
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			log.Fatalf("не удалось прочитать переменные окружения: %v", err)
		}
	})
	return &cfg
}
