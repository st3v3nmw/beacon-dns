package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/st3v3nmw/beacon/internal/dns"
)

func trace(c echo.Context) error {
	nameParam := c.QueryParam("name")
	if nameParam == "" {
		err := "`name` query param must be provided"
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	trace, err := dns.HandleTrace(nameParam, c.RealIP())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, trace)
}
