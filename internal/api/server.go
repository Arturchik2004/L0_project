package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	fs := http.FileServer(http.Dir("./web/"))
	r.Handle("/*", fs)

	r.Route("/api", func(r chi.Router) {
		r.Get("/order/{orderUID}", h.GetOrder)
		r.Get("/orders/recent", h.GetRecentOrders)
	})

	return r
}

func StartServer(port string, router *chi.Mux) error {
	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("Сервер запущен на http://localhost%s\n", addr)
	return http.ListenAndServe(addr, router)
}
