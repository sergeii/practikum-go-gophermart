package handlers

import "github.com/sergeii/practikum-go-gophermart/internal/application"

type Handler struct {
	app *application.App
}

func New(app *application.App) *Handler {
	return &Handler{app}
}
