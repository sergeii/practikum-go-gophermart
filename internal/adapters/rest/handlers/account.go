package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/internal/adapters/rest/middleware/auth"
	"github.com/sergeii/practikum-go-gophermart/internal/core/users"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
	"github.com/sergeii/practikum-go-gophermart/internal/services/account"
)

type RegisterUserReq struct {
	Login    string `json:"login" binding:"required,notblank"`
	Password string `json:"password" binding:"required,notblank"`
}

type RegisterUserResp struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
}

func (h *Handler) RegisterUser(c *gin.Context) {
	var json RegisterUserReq
	if err := c.ShouldBindJSON(&json); err != nil {
		log.Debug().Err(err).Str("path", c.FullPath()).Msg("unable to parse register request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.app.UserService.RegisterNewUser(
		c.Request.Context(),
		strings.TrimSpace(json.Login),
		strings.TrimSpace(json.Password),
	)
	if err != nil {
		if errors.Is(err, users.ErrUserLoginIsOccupied) {
			log.Debug().
				Err(err).Str("path", c.FullPath()).Str("login", json.Login).
				Msg("unable to register user due to conflict")
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			log.Error().
				Err(err).Str("path", c.FullPath()).Str("login", json.Login).
				Msg("unable to register user due to error")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	log.Info().
		Str("path", c.FullPath()).Int("id", u.ID).Str("login", u.Login).
		Msg("registered new user")

	if err := h.setAuthCookie(c, u); err != nil {
		log.Error().
			Err(err).Str("path", c.FullPath()).Str("login", json.Login).
			Msg("failed to set auth cookie")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": RegisterUserResp{ID: u.ID, Login: u.Login}})
}

type LoginUserReq struct {
	Login    string `json:"login" binding:"required,notblank"`
	Password string `json:"password" binding:"required,notblank"`
}

func (h *Handler) LoginUser(c *gin.Context) {
	var json LoginUserReq

	if err := c.ShouldBindJSON(&json); err != nil {
		log.Debug().Err(err).Str("path", c.FullPath()).Msg("unable to parse login request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.app.UserService.Authenticate(c.Request.Context(), json.Login, json.Password)
	if err != nil {
		switch {
		case errors.Is(err, account.ErrAuthenticateInvalidCredentials):
			log.Debug().
				Err(err).Str("path", c.FullPath()).Str("login", json.Login).
				Msg("unable to login user due to login/password mismatch")
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		case errors.Is(err, account.ErrAuthenticateEmptyPassword):
			log.Debug().Err(err).Str("path", c.FullPath()).Msg("unable to login user with empty password")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			log.Error().
				Err(err).Str("path", c.FullPath()).Str("login", json.Login).
				Msg("unable to register user due to error")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	log.Info().
		Str("path", c.FullPath()).Int("id", u.ID).Str("login", u.Login).
		Msg("user logged in")

	if err := h.setAuthCookie(c, u); err != nil {
		log.Error().
			Err(err).Str("path", c.FullPath()).Str("login", json.Login).
			Msg("failed to set auth cookie")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": RegisterUserResp{ID: u.ID, Login: u.Login}})
}

func (h *Handler) setAuthCookie(c *gin.Context, u models.User) error {
	token, err := auth.GenerateAuthTokenCookie(u, h.app.Cfg.SecretKey)
	if err != nil {
		return err
	}
	cookie := http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(auth.CookieAge),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(c.Writer, &cookie)
	return nil
}
