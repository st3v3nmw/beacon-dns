package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/st3v3nmw/beacon/internal/dns"
)

func queryDNS(c echo.Context) error {
	filter, err := dns.NewFilterFromStr(c.Param("filter"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

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

	response, err := dns.HandleDoHRequest(qn, filter)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, response)
}
