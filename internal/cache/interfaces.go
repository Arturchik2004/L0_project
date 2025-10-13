package cache

import "L0_project/internal/model"

// OrderCache определяет интерфейс для кэша заказов.
type OrderCache interface {
	Add(key string, order *model.Order)
	Get(key string) (*model.Order, bool)
}
