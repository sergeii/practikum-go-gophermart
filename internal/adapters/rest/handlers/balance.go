package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/internal/adapters/rest/middleware/auth"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
	"github.com/sergeii/practikum-go-gophermart/pkg/encode"
)

type UserBalanceResp struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

func (h *Handler) ShowUserBalance(c *gin.Context) {
	u := c.MustGet(auth.ContextKey).(models.User) // nolint: forcetypeassert
	balance, err := h.app.UserService.GetBalance(c.Request.Context(), u.ID)
	if err != nil {
		log.Error().
			Err(err).Str("path", c.FullPath()).Int("userID", u.ID).
			Msg("Unable to show user balance due to error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	c.JSON(http.StatusOK, UserBalanceResp{
		encode.DecimalToFloat(balance.Current),
		encode.DecimalToFloat(balance.Withdrawn),
	})
}
