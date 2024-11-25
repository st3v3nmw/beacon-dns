package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/st3v3nmw/beacon/internal/dns"
)

func trace(c echo.Context) error {
	name := c.QueryParam("name")
	if name == "" {
		err := "`name` query param must be provided"
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	trace, err := dns.HandleTrace(name, c.RealIP())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, trace)
}