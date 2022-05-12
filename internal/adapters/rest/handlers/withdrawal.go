package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/sergeii/practikum-go-gophermart/internal/adapters/rest/middleware/auth"
	"github.com/sergeii/practikum-go-gophermart/internal/core/users"
	"github.com/sergeii/practikum-go-gophermart/internal/services/withdrawal"
	"github.com/sergeii/practikum-go-gophermart/pkg/encode"
)

type WithdrawalReq struct {
	Order string  `json:"order" binding:"required,numeric,luhn"`
	Sum   float64 `json:"sum" binding:"required,gt=0"`
}

type WithdrawalResp struct {
	ID          int       `json:"id"`
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"` // nolint: tagliatelle
}

func (h *Handler) RequestWithdrawal(c *gin.Context) {
	user := c.MustGet(auth.ContextKey).(users.User) // nolint: forcetypeassert

	var json WithdrawalReq
	if err := c.ShouldBindJSON(&json); err != nil {
		log.Debug().
			Err(err).Str("path", c.FullPath()).Int("userID", user.ID).
			Msg("Unable to validate withdrawal request")
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	w, err := h.app.WithdrawalService.RequestWithdrawal(
		c.Request.Context(), json.Order, user.ID, decimal.NewFromFloat(json.Sum),
	)
	if err != nil {
		log.Warn().
			Err(err).Str("path", c.FullPath()).
			Str("order", json.Order).Float64("sum", json.Sum).Int("userID", user.ID).
			Msg("Failed to request withdrawal")
		switch {
		case errors.Is(err, withdrawal.ErrWithdrawalAlreadyRegistered):
			c.JSON(http.StatusConflict, gin.H{"error": "withdrawal with this order has already been registered"})
		case errors.Is(err, withdrawal.ErrWithdrawalInvalidSumSum):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, users.ErrUserHasInsufficientBalance):
			c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	result := WithdrawalResp{
		ID:          w.ID,
		Order:       w.Number,
		Sum:         encode.DecimalToFloat(w.Sum),
		ProcessedAt: w.ProcessedAt,
	}
	c.JSON(http.StatusOK, gin.H{"result": result})
}

type ListWithdrawalRespItem struct {
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"` // nolint: tagliatelle
}

func (h *Handler) ListUserWithdrawals(c *gin.Context) {
	user := c.MustGet(auth.ContextKey).(users.User) // nolint: forcetypeassert
	userWithdrawals, err := h.app.WithdrawalService.GetUserWithdrawals(c.Request.Context(), user.ID)

	if err != nil {
		log.Warn().
			Err(err).Str("path", c.FullPath()).Int("userID", user.ID).
			Msg("Unable to fetch withdrawals for user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(userWithdrawals) == 0 {
		c.Status(http.StatusNoContent)
		return
	}
	jsonItems := make([]ListWithdrawalRespItem, 0, len(userWithdrawals))
	for _, w := range userWithdrawals {
		jsonItems = append(jsonItems, ListWithdrawalRespItem{
			w.Number,
			encode.DecimalToFloat(w.Sum),
			w.ProcessedAt,
		})
	}
	c.JSON(http.StatusOK, jsonItems)
}
