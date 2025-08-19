package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/asquebay/simple-order-service/internal/model"
	"github.com/asquebay/simple-order-service/internal/repository/postgres"
)

// OrderGetter определяет интерфейс для сервиса, который может получать заказы
// Это позволяет хэндлеру не зависеть от конкретной реализации сервиса
type OrderGetter interface {
	GetOrderByUID(ctx context.Context, uid string) (model.Order, error)
}

// Handler обрабатывает HTTP-запросы
type Handler struct {
	service OrderGetter
	log     *slog.Logger
	mux     *http.ServeMux
}

// NewHandler создает новый экземпляр Handler
func NewHandler(service OrderGetter, log *slog.Logger) *Handler {
	h := &Handler{
		service: service,
		log:     log,
		mux:     http.NewServeMux(),
	}
	h.registerRoutes()
	return h
}

// ServeHTTP делает Handler совместимым с http.Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// registerRoutes регистрирует все эндпоинты
func (h *Handler) registerRoutes() {
	// роутинг для получения заказа по ID
	h.mux.HandleFunc("GET /order/{order_uid}", h.getOrderByUID)

	// роутинг для статики (HTML/JS/CSS)
	fileServer := http.FileServer(http.Dir("./web/"))
	h.mux.Handle("/", http.StripPrefix("/", fileServer))
}

func (h *Handler) getOrderByUID(w http.ResponseWriter, r *http.Request) {
	// извлекаем order_uid из URL
	uid := r.PathValue("order_uid")
	if uid == "" {
		h.respondError(w, http.StatusBadRequest, "order_uid is required")
		return
	}

	order, err := h.service.GetOrderByUID(r.Context(), uid)
	if err != nil {
		if errors.Is(err, postgres.ErrOrderNotFound) {
			h.respondError(w, http.StatusNotFound, "order not found")
			return
		}
		h.log.Error("internal server error", slog.String("error", err.Error()))
		h.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.respondJSON(w, http.StatusOK, order)
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		h.log.Error("failed to marshal JSON response", slog.String("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write(response)
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
