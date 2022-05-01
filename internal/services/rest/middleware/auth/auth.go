package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/sergeii/practikum-go-gophermart/internal/models"
)

const AuthCookieName = "auth"
const AuthCookieAge = time.Hour * 24 * 365

type TokenClaims struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
	jwt.RegisteredClaims
}

func GenerateAuthTokenCookie(user models.User, secretKey []byte) (string, error) {
	claims := TokenClaims{
		user.ID,
		user.Login,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AuthCookieAge)),
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
