package database

import (
	"L0_project/internal/model"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Storage struct {
	db *sqlx.DB
}

func New(databaseURL string) (*Storage, error) {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("не удалось проверить соединение с базой данных: %w", err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) ApplyMigrations(path string) error {
	c, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("не удалось прочитать файл схемы: %w", err)
	}
	schema := string(c)
	_, err = s.db.ExecContext(context.Background(), schema)
	if err != nil {
		return fmt.Errorf("не удалось применить migrations: %w", err)
	}
	log.Println("Migrations успешно применены")
	return nil
}

func (s *Storage) SaveOrder(ctx context.Context, order *model.Order) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("не удалось начать транзакцию: %w", err)
	}
	defer tx.Rollback()

	var deliveryID int
	deliveryQuery := `INSERT INTO deliveries (name, phone, zip, city, address, region, email)
                     VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	err = tx.QueryRowxContext(ctx, deliveryQuery, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip, order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email).Scan(&deliveryID)
	if err != nil {
		return fmt.Errorf("не удалось вставить данные доставки: %w", err)
	}

	var paymentID int
	paymentQuery := `INSERT INTO payments (transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
                   VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`
	err = tx.QueryRowxContext(ctx, paymentQuery, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency, order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank, order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee).Scan(&paymentID)
	if err != nil {
		return fmt.Errorf("не удалось вставить данные оплаты: %w", err)
	}

	orderQuery := `INSERT INTO orders (order_uid, track_number, entry, delivery_id, payment_id, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
                 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err = tx.ExecContext(ctx, orderQuery, order.OrderUID, order.TrackNumber, order.Entry, deliveryID, paymentID, order.Locale, order.InternalSignature, order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		return fmt.Errorf("не удалось вставить данные заказа: %w", err)
	}

	for _, item := range order.Items {
		itemQuery := `INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
                    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
		_, err = tx.ExecContext(ctx, itemQuery, order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name, item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			return fmt.Errorf("не удалось вставить товар для заказа %s: %w", order.OrderUID, err)
		}
	}

	return tx.Commit()
}

func (s *Storage) GetOrder(ctx context.Context, orderUID string) (*model.Order, error) {
	var order model.Order

	query := `
        SELECT
            o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature, o.customer_id, o.delivery_service,
            o.shardkey, o.sm_id, o.date_created, o.oof_shard,
            d.name "delivery.name", d.phone "delivery.phone", d.zip "delivery.zip", d.city "delivery.city",
            d.address "delivery.address", d.region "delivery.region", d.email "delivery.email",
            p.transaction "payment.transaction", p.request_id "payment.request_id", p.currency "payment.currency",
            p.provider "payment.provider", p.amount "payment.amount", p.payment_dt "payment.payment_dt", p.bank "payment.bank",
            p.delivery_cost "payment.delivery_cost", p.goods_total "payment.goods_total", p.custom_fee "payment.custom_fee"
        FROM orders o
        JOIN deliveries d ON o.delivery_id = d.id
        JOIN payments p ON o.payment_id = p.id
        WHERE o.order_uid = $1`

	err := s.db.GetContext(ctx, &order, query, orderUID)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить заказ %s: %w", orderUID, err)
	}

	itemsQuery := `SELECT * FROM items WHERE order_uid = $1`
	err = s.db.SelectContext(ctx, &order.Items, itemsQuery, orderUID)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить товары для заказа %s: %w", orderUID, err)
	}

	return &order, nil
}

func (s *Storage) GetAllOrders(ctx context.Context) ([]model.Order, error) {
	var orderUIDs []string
	err := s.db.SelectContext(ctx, &orderUIDs, "SELECT order_uid FROM orders ORDER BY date_created DESC")
	if err != nil {
		return nil, fmt.Errorf("не удалось получить все идентификаторы заказов: %w", err)
	}

	var orders []model.Order
	for _, uid := range orderUIDs {
		order, err := s.GetOrder(ctx, uid)
		if err != nil {
			log.Printf("Ошибка загрузки заказа %s для прогрева кеша: %v", uid, err)
			continue
		}
		orders = append(orders, *order)
	}

	return orders, nil
}

func (s *Storage) GetRecentOrders(ctx context.Context, limit int) ([]model.Order, error) {
	var orderUIDs []string
	query := `SELECT order_uid FROM orders ORDER BY date_created DESC LIMIT $1`
	err := s.db.SelectContext(ctx, &orderUIDs, query, limit)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить последние идентификаторы заказов: %w", err)
	}

	var orders []model.Order
	for _, uid := range orderUIDs {
		order, err := s.GetOrder(ctx, uid)
		if err != nil {
			log.Printf("Ошибка загрузки заказа %s для списка последних заказов: %v", uid, err)
			continue
		}
		orders = append(orders, *order)
	}

	return orders, nil
}

func (s *Storage) Close() {
	s.db.Close()
}
