package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/internal/adapters/rest/middleware/auth"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
	"github.com/sergeii/practikum-go-gophermart/internal/ports/queue"
	"github.com/sergeii/practikum-go-gophermart/internal/services/order"
	"github.com/sergeii/practikum-go-gophermart/pkg/encode"
)

type UploadOrderResp struct {
	ID         int                `json:"id"`
	Number     string             `json:"number"`
	Status     models.OrderStatus `json:"status"`
	UploadedAt time.Time          `json:"uploaded_at"` // nolint: tagliatelle
}

// not an actual request schema, it is used merely for validation purposed
type uploadOrderField struct {
	Number string `binding:"required,numeric,luhn"`
}

func (h *Handler) UploadOrder(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		log.Error().
			Err(err).Str("path", c.FullPath()).
			Msg("Encountered an error while obtaining order number")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// the order number is passed in body in plaintext
	orderNumber := strings.TrimSpace(string(body))
	if orderNumber == "" {
		log.Debug().Err(err).Str("path", c.FullPath()).Msg("Missing order number")
		c.JSON(http.StatusBadRequest, gin.H{"error": "order number is required"})
		return
	}
	// the order number should be a numeric value
	// also it should pass luhn validation
	if err = binding.Validator.ValidateStruct(uploadOrderField{orderNumber}); err != nil {
		log.Debug().
			Err(err).Str("path", c.FullPath()).Str("number", orderNumber).
			Msg("Invalid order number format")
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "order number does not conform the format"})
		return
	}

	user := c.MustGet(auth.ContextKey).(models.User) // nolint: forcetypeassert
	o, err := h.app.OrderService.SubmitNewOrder(c.Request.Context(), orderNumber, user.ID)
	if err != nil {
		log.Warn().
			Err(err).Str("path", c.FullPath()).Str("number", orderNumber).
			Msg("Unable to upload new order")
		switch {
		case errors.Is(err, order.ErrOrderUploadedByAnotherUser):
			c.JSON(http.StatusConflict, gin.H{"error": "order has already been uploaded by another user"})
		case errors.Is(err, order.ErrOrderAlreadyUploaded):
			c.Status(http.StatusOK)
		case errors.Is(err, queue.ErrQueueIsFull):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	result := UploadOrderResp{
		ID:         o.ID,
		Number:     o.Number,
		Status:     o.Status,
		UploadedAt: o.UploadedAt,
	}
	c.JSON(http.StatusAccepted, gin.H{"result": result})
}

type ListOrderRespItem struct {
	Number     string             `json:"number"`
	Status     models.OrderStatus `json:"status"`
	Accrual    float64            `json:"accrual"`
	UploadedAt time.Time          `json:"uploaded_at"` // nolint: tagliatelle
}

func (h *Handler) ListUserOrders(c *gin.Context) {
	user := c.MustGet(auth.ContextKey).(models.User) // nolint: forcetypeassert
	orders, err := h.app.OrderService.GetUserOrders(c.Request.Context(), user.ID)
	if err != nil {
		log.Warn().
			Err(err).Str("path", c.FullPath()).Int("userID", user.ID).
			Msg("Unable to fetch orders for user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(orders) == 0 {
		c.Status(http.StatusNoContent)
		return
	}
	jsonItems := make([]ListOrderRespItem, 0, len(orders))
	for _, o := range orders {
		jsonItems = append(jsonItems, ListOrderRespItem{
			o.Number,
			o.Status,
			encode.DecimalToFloat(o.Accrual),
			o.UploadedAt,
		})
	}
	c.JSON(http.StatusOK, jsonItems)
}
