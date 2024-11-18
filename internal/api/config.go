package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/st3v3nmw/beacon/internal/config"
)

func getConfig(c echo.Context) error {
	return c.JSON(http.StatusOK, config.All)
}
