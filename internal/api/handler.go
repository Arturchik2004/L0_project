package api

import (
	"L0_project/internal/cache"
	"L0_project/internal/database"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	db    *database.Storage
	cache cache.Cache
}

func NewHandler(db *database.Storage, cache cache.Cache) *Handler {
	return &Handler{db: db, cache: cache}
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderUID := chi.URLParam(r, "orderUID")
	if orderUID == "" {
		http.Error(w, "Идентификатор заказа обязателен", http.StatusBadRequest)
		return
	}

	if order, found := h.cache.Get(orderUID); found {
		log.Printf("Cache HIT для заказа: %s", orderUID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(order)
		return
	}

	log.Printf("Cache MISS для заказа: %s. Загрузка из базы данных.", orderUID)
	order, err := h.db.GetOrder(r.Context(), orderUID)
	if err != nil {
		log.Printf("Ошибка получения заказа из базы данных: %v", err)
		http.Error(w, "Заказ не найден", http.StatusNotFound)
		return
	}

	h.cache.Set(orderUID, order)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (h *Handler) GetRecentOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.db.GetRecentOrders(r.Context(), 10)
	if err != nil {
		log.Printf("Ошибка получения последних заказов из базы данных: %v", err)
		http.Error(w, "Не удалось получить последние заказы", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}
