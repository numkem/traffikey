package main

import (
	"github.com/labstack/echo/v4"
)

type handler struct {
	monitor *Monitor
}

func (h *handler) List(c echo.Context) error {
	return c.JSON(200, nil)
}
