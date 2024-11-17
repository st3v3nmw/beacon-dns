package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/st3v3nmw/beacon/internal/querylog"
)

func getStats(c echo.Context) error {
	stats, err := querylog.GetDeviceStats()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, stats)
}
