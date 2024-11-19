package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/st3v3nmw/beacon/internal/dns"
)

const (
	dnsJsonDataType   = "application/dns-json"
	dnsWireFormatType = "application/dns-message"
)

func trace(c echo.Context) error {
	name := c.QueryParam("name")
	if name == "" {
		err := "`name` query param must be provided"
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	qtype := c.QueryParam("type")
	if qtype == "" {
		err := "`qtype` query param must be provided"
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	trace, err := dns.HandleTrace(name, qtype)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, trace)
}
