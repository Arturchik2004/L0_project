# L0_project

*Краткое описание*

Проект — учебный сервис для приёма и обработки заказов. 

### Основные компоненты:
- Продюсер (`cmd/producer`) — генерирует сообщения о заказах и отправляет их в Kafka.
- Консьюмер (`internal/kafka`) — читает сообщения из Kafka, валидирует, сохраняет в PostgreSQL и кэширует в LRU.
- HTTP API (`cmd/main` + `internal/api`) — предоставляет эндпоинты для получения данных о заказах.

### Структура проекта

- `cmd/` — исполняемые команды
  - `main/` — основной HTTP-сервис и точка входа приложения
  - `producer/` — генератор заказов и отправщик в Kafka
- `internal/` — внутренняя логика
  - `api/` — HTTP-роутер и хендлеры
  - `cache/` — LRU cache реализация
  - `config/` — чтение конфигурации через переменные окружения
  - `database/` — реализация работы с PostgreSQL (Storage) и схема миграций
  - `kafka/` — consumer логика
  - `model/` — структуры данных (Order, Delivery, Payment, Item)
- `web/` — статические файлы фронтенда
- `migrations/` — SQL-файлы миграций (`000001_init.up.sql`, `000001_init.down.sql`)
- `.env.example` — пример переменных окружения

### Архитектура и поток данных

1. Продюсер генерирует JSON заказа и публикует в Kafka topic.
2. Консьюмер читает сообщения и выполняет валидацию данных (структурную и бизнес-правила).
3. После успешной валидации запись сохраняется в хранилище (Postgres) через `OrderStorage` — интерфейс, позволяющий менять реализацию (например, мок в тестах).
4. Данные кэшируются в LRU для уменьшения нагрузки на БД.
5. HTTP API читает из кеша/БД и возвращает данные клиенту.

### Используемые технологии

- Go, Docker и Docker Compose
- Kafka (Confluent, topic-based architecture)
- PostgreSQL
- go-chi (HTTP router)
- sqlx (DB helper)
- github.com/brianvoe/gofakeit/v7 (генерация реалистичных тестовых данных)
- github.com/go-playground/validator/v10 (валидация входных данных)

### Запуск локально

1) Скопировать пример `.env` и отредактировать при необходимости:

```powershell
Copy-Item .env.example .env
```

2) Поднять инфраструктуру через Docker Compose:
```powershell
docker compose up -d
docker compose ps
```

3) Применить миграции (рекомендуемый способ — использовать `golang-migrate`):

```powershell
docker run --rm -v "${PWD}\migrations:/migrations" --network l0_project_default migrate/migrate -path=/migrations -database "postgres://postgres:123@postgres-db:5432/GoLangWB?sslmode=disable" up
```

4) Запустить сервисы:

- *Запустите основной сервис (в новом терминале):*
```powershell
go run ./cmd/main
```
- *Запустите генератор заказов (в новом терминале):*
```powershell
go run ./cmd/producer
```

### Режимы producer

- По умолчанию продюсер генерирует данные через `gofakeit`.
- Чтобы использовать `model.json` как шаблон, выставьте переменную окружения:

```powershell
$env:PRODUCER_MODE = 'json'
go run ./cmd/producer
```

## Валидация

- Проект использует `github.com/go-playground/validator/v10` для валидации данных на основе тегов в структурах модели (см. `internal/model/order.go`).
- Валидация происходит в консьюмере перед сохранением в базу данных.
```go
import (
  ...
	"github.com/go-playground/validator/v10"
	...
)

type Consumer struct {
...
	validate *validator.Validate
}

func NewConsumer(brokers []string, topic, groupID string, db database.OrderStorage, cache cache.OrderCache) *Consumer {
	...
	return &Consumer{reader: r, db: db, cache: cache, validate: validator.New()}
}
```

## Ошибки и логирование
- Логирование реализовано через стандартный `log` пакет.
- Ошибки обработки сообщений в Kafka не приводят к остановке сервиса, а логируются. Невалидные сообщения или сообщения, вызвавшие ошибку сохранения, пропускаются, чтобы не блокировать очередь.

```go
// Фрагмент из `internal/kafka/consumer.go`
func (c *Consumer) Start(ctx context.Context) {
	// ...
	for {
		// ...
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			log.Printf("не удалось получить сообщение: %v", err)
			continue
		}

		var order model.Order
		if err := json.Unmarshal(m.Value, &order); err != nil {
			log.Printf("не удалось разобрать сообщение: %v. Сообщение: %s", err, string(m.Value))
			c.reader.CommitMessages(ctx, m) // Пропускаем невалидное сообщение
			continue
		}

		if err := c.validate.Struct(&order); err != nil {
			log.Printf("невалидные данные в заказе %s: %v", order.OrderUID, err)
			c.reader.CommitMessages(ctx, m) // Пропускаем невалидное сообщение
			continue
		}

		if err := c.db.SaveOrder(ctx, &order); err != nil {
			log.Printf("не удалось сохранить заказ %s в базу данных: %v", order.OrderUID, err)
			// Здесь сообщение не подтверждается, будет повторная попытка
			continue
		}
		// ...
		c.reader.CommitMessages(ctx, m) // Подтверждаем успешную обработку
	}
}
```

### Миграции

- Миграции хранятся в папке `migrations/` в формате up/down. Приложение не должно автоматически накатывать миграции в проде — используйте `golang-migrate` или CI-пайплайн для управления миграциями.

### Интерфейсы и тестируемость

- Ключевые зависимости абстрагированы через интерфейсы (например, `database.OrderStorage`) — это упрощает написание моков и тестирование.




