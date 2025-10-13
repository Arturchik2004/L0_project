package database

import (
	"L0_project/internal/model"
	"context"
)

// OrderStorage описывает минимальный набор операций для работы с заказами
type OrderStorage interface {
	SaveOrder(ctx context.Context, order *model.Order) error
	GetOrder(ctx context.Context, orderUID string) (*model.Order, error)
	GetAllOrders(ctx context.Context) ([]model.Order, error)
	GetRecentOrders(ctx context.Context, limit int) ([]model.Order, error)
}
