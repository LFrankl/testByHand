// HTTP 层：秒杀接口完整实现
//
// POST /seckill?user_id=123
//   成功 → 200  {"order_id": "ORD-..."}
//   库存不足 → 409  {"error": "out of stock"}
//   重复购买 → 409  {"error": "already bought"}
//
// GET /stock
//   → 200  {"stock": 42}
package seckill

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

// Handler 持有 SeckillService 并实现 http.Handler。
type Handler struct {
	svc *SeckillService
}

// NewHandler 创建 HTTP 处理器。
func NewHandler(svc *SeckillService) *Handler {
	return &Handler{svc: svc}
}

// ServeHTTP 路由分发。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/seckill":
		h.handleSeckill(w, r)
	case "/stock":
		h.handleStock(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleSeckill 处理秒杀请求。
func (h *Handler) handleSeckill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userID, err := parseUserID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	orderID, err := h.svc.Seckill(userID)
	if err != nil {
		// ErrOutOfStock / ErrAlreadyBought 统一返回 409 Conflict
		if errors.Is(err, ErrOutOfStock) || errors.Is(err, ErrAlreadyBought) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"order_id": orderID})
}

// handleStock 返回当前库存。
func (h *Handler) handleStock(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]int{"stock": h.svc.Stock()})
}

// -------- 工具函数 --------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func parseUserID(r *http.Request) (int64, error) {
	raw := r.URL.Query().Get("user_id")
	if raw == "" {
		return 0, errors.New("missing user_id")
	}
	return strconv.ParseInt(raw, 10, 64)
}
