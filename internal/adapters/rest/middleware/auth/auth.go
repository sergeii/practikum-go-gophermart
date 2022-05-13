package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/internal/core/users"
)

const CookieName = "auth"
const CookieAge = time.Hour * 24 * 365

const ContextKey = "auth"

var ErrInvalidSigningMethod = errors.New("invalid signing method")

type TokenClaims struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
	jwt.RegisteredClaims
}

func GenerateAuthTokenCookie(user users.User, secretKey []byte) (string, error) {
	claims := TokenClaims{
		user.ID,
		user.Login,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(CookieAge)),
			Issuer:    "gophermart",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func Authentication(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer c.Next()
		cookie, err := c.Cookie(CookieName)
		if err != nil {
			return
		}

		token, err := jwt.ParseWithClaims(cookie, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidSigningMethod
			}
			return cfg.SecretKey, nil
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse jwt token")
			return
		}

		if claims, ok := token.Claims.(*TokenClaims); token.Valid && ok {
			user := users.NewFromID(claims.ID)
			log.Debug().
				Int("userID", user.ID).Str("login", user.Login).
				Msg("Successfully authenticated user")
			c.Set(ContextKey, user)
		} else {
			log.Warn().Msg("Failed to obtain token claims")
		}
	}
}

func RequireAuthentication(c *gin.Context) {
	if _, ok := c.Get(ContextKey); !ok {
		log.Debug().Str("path", c.FullPath()).Msg("Endpoint is for authenticated users only")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
	c.Next()
}
