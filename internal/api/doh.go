package api

import (
	"encoding/base64"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/st3v3nmw/beacon/internal/dns"
)

const (
	dnsJsonDataType   = "application/dns-json"
	dnsWireFormatType = "application/dns-message"
)

func queryDNS(c echo.Context) error {
	filter, err := dns.NewFilterFromStr(c.Param("filter"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	h := c.Request().Header
	if h.Get("Accept") == dnsWireFormatType || h.Get("Content-Type") == dnsWireFormatType {
		return processDoHWire(c, filter)
	} else {
		return processDoHJson(c, filter)
	}
}

func processDoHWire(c echo.Context, filter *dns.Filter) error {
	var query []byte
	var err error

	r := c.Request()
	if r.Method == "GET" {
		dnsQueryParam := r.URL.Query().Get("dns")
		if dnsQueryParam == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "The `dns` query param must be provided")
		}

		query, err = base64.RawURLEncoding.DecodeString(dnsQueryParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
	} else {
		query, err = io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body")
		}
		defer c.Request().Body.Close()
	}

	response, err := dns.HandleDoHReqWire(query, filter)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.Blob(http.StatusOK, dnsWireFormatType, response)
}

func processDoHJson(c echo.Context, filter *dns.Filter) error {
	var err error
	qn := &dns.Request{
		Type: dns.QType(1), // default to A
	}

	if c.Request().Method == "GET" {
		err = echo.QueryParamsBinder(c).
			String("name", &qn.Name).
			JSONUnmarshaler("type", &qn.Type).
			Bool("do", &qn.DO).
			Bool("cd", &qn.CD).
			Bool("trace", &qn.Trace).
			BindError()
	} else {
		s := echo.DefaultJSONSerializer{}
		err = s.Deserialize(c, qn)
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	err = c.Validate(*qn)
	if err != nil {
		return err
	}

	response, err := dns.HandleDoHReqJson(qn, filter)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, response)
}
