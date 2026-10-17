package rest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"thesis/app/internal/config"
	"thesis/app/internal/domain"
	"thesis/app/internal/metrics"
	"thesis/app/internal/security"
)

type API struct {
	svc domain.Service
	cfg config.Config
	val *validator.Validate
}

type newItemReq struct {
	ProductID int64 `json:"productId" validate:"required,gt=0"`
	Quantity  int32 `json:"quantity" validate:"required,gte=1,lte=1000"`
}

type newOrderReq struct {
	UserID int64        `json:"userId" validate:"required,gt=0"`
	Items  []newItemReq `json:"items" validate:"required,min=1,dive"`
}

var statuses = map[string]bool{"NEW": true, "PAID": true, "SHIPPED": true, "CANCELLED": true}

// New buduje router. Kolejność middleware: metryki -> JWT -> rate limit -> rozmiar (walidacja) -> handler.
func New(svc domain.Service, cfg config.Config) http.Handler {
	a := &API{svc: svc, cfg: cfg, val: validator.New()}
	r := chi.NewRouter()
	r.Use(metrics.HTTP("rest", func(r *http.Request) string {
		if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
			return r.Method + " " + rc.RoutePattern()
		}
		return "other"
	}))
	if cfg.Auth {
		r.Use(security.Auth(security.NewJWT(cfg.JWTSecret), "/healthz"))
	}
	if cfg.RateLimit {
		r.Use(security.RateLimit(security.NewLimiters(cfg.RateLimitRPS, cfg.RateLimitBurst), "/healthz"))
	}
	if cfg.Validation {
		r.Use(security.MaxBody(cfg.MaxBodyBytes))
	}
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/users/{id}", a.getUser)
	r.Get("/users/{id}/orders", a.getUserOrders)
	r.Get("/orders", a.listOrders)
	r.Post("/orders", a.createOrder)
	return r
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, domain.ErrInvalid):
		http.Error(w, "invalid argument", http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (a *API) intParam(w http.ResponseWriter, s string, def int) (int, bool) {
	if s == "" {
		return def, true
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		http.Error(w, "invalid argument", http.StatusBadRequest)
		return 0, false
	}
	return n, true
}

func (a *API) checkLimit(w http.ResponseWriter, limit int) bool {
	if a.cfg.Validation && (limit < 1 || limit > domain.MaxLimit) {
		http.Error(w, "invalid limit", http.StatusBadRequest)
		return false
	}
	return true
}

func (a *API) getUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || (a.cfg.Validation && id <= 0) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	u, err := a.svc.GetUser(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (a *API) getUserOrders(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || (a.cfg.Validation && id <= 0) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	limit, ok := a.intParam(w, r.URL.Query().Get("limit"), domain.DefaultOrdersLimit)
	if !ok || !a.checkLimit(w, limit) {
		return
	}
	// limit parametru expand (C5): dozwolone tylko items.product
	if exp := r.URL.Query().Get("expand"); a.cfg.Complexity && exp != "" && exp != "items.product" {
		http.Error(w, "unsupported expand", http.StatusBadRequest)
		return
	}
	res, err := a.svc.GetUserWithOrders(r.Context(), id, limit)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (a *API) listOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, ok := a.intParam(w, q.Get("limit"), domain.DefaultListLimit)
	if !ok || !a.checkLimit(w, limit) {
		return
	}
	offset, ok := a.intParam(w, q.Get("offset"), 0)
	if !ok {
		return
	}
	status := q.Get("status")
	if a.cfg.Validation && (offset < 0 || (status != "" && !statuses[status])) {
		http.Error(w, "invalid argument", http.StatusBadRequest)
		return
	}
	res, err := a.svc.ListOrders(r.Context(), status, limit, offset)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (a *API) createOrder(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if a.cfg.Complexity {
		if err := security.CheckJSON(data, a.cfg.MaxJSONDepth, a.cfg.MaxArrayLen); err != nil {
			http.Error(w, "payload too complex", http.StatusBadRequest)
			return
		}
	}
	var req newOrderReq
	if err := json.Unmarshal(data, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if a.cfg.Validation {
		if err := a.val.Struct(req); err != nil {
			http.Error(w, "validation failed", http.StatusBadRequest)
			return
		}
	}
	in := domain.NewOrder{UserID: req.UserID}
	for _, it := range req.Items {
		in.Items = append(in.Items, domain.NewItem{ProductID: it.ProductID, Quantity: it.Quantity})
	}
	o, err := a.svc.CreateOrder(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, o)
}
