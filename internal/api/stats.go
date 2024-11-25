package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/st3v3nmw/beacon/internal/dns"
	"github.com/st3v3nmw/beacon/internal/querylog"
)

func getDeviceStats(c echo.Context) error {
	stats, err := querylog.GetDeviceStats()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, stats)
}

func getCacheStats(c echo.Context) error {
	return c.JSON(http.StatusOK, dns.GetCacheStats())
}
