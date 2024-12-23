package api

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/st3v3nmw/beacon/internal/dns"
	"github.com/st3v3nmw/beacon/internal/querylog"
)

func getDeviceStats(c echo.Context) error {
	lastParam := c.QueryParam("last")
	if lastParam == "" {
		lastParam = "24h"
	}

	last, err := time.ParseDuration(lastParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	stats, err := querylog.GetDeviceStats(last)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, stats)
}

func getCacheStats(c echo.Context) error {
	return c.JSON(http.StatusOK, dns.GetCacheStats())
}
