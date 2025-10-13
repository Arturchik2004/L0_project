package database

import (
	"L0_project/internal/model"
	"context"
	"fmt"
)

// MockStorage простой мок для тестов
type MockStorage struct {
	Orders map[string]model.Order
}

func NewMockStorage() *MockStorage {
	return &MockStorage{Orders: make(map[string]model.Order)}
}

func (m *MockStorage) SaveOrder(ctx context.Context, order *model.Order) error {
	m.Orders[order.OrderUID] = *order
	return nil
}

func (m *MockStorage) GetOrder(ctx context.Context, orderUID string) (*model.Order, error) {
	if o, ok := m.Orders[orderUID]; ok {
		return &o, nil
	}
	return nil, ErrNotFound
}

func (m *MockStorage) GetAllOrders(ctx context.Context) ([]model.Order, error) {
	var res []model.Order
	for _, o := range m.Orders {
		res = append(res, o)
	}
	return res, nil
}

func (m *MockStorage) GetRecentOrders(ctx context.Context, limit int) ([]model.Order, error) {
	return m.GetAllOrders(ctx)
}

// ErrNotFound используется в моках
var ErrNotFound = fmt.Errorf("not found")
