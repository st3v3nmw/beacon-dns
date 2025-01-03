package api

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

var (
	API *APIService
)

type APIService struct {
	address string
	echo    *echo.Echo
}

func New(addr string) {
	e := echo.New()

	e.Validator = &customValidator{validator: validator.New()}

	// Home
	e.GET("/api", func(c echo.Context) error {
		return c.String(http.StatusOK, "Beacon DNS API")
	})

	// Config
	e.GET("/api/config", getConfig)

	// Watching
	e.GET("/api/watch", watch)

	// Statistics
	e.GET("/api/stats/devices", getDeviceStats)
	e.GET("/api/stats/cache", getCacheStats)

	// Trace
	e.GET("/api/trace", trace)

	API = &APIService{
		address: addr,
		echo:    e,
	}
}

func Start() error {
	return API.echo.Start(API.address)
}

// validator
type customValidator struct {
	validator *validator.Validate
}

func (cv *customValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}
