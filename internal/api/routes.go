package api

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
	}))

	e.Validator = &customValidator{validator: validator.New()}

	e.GET("/", home)

	e.GET("/watch", watch)

	e.GET("/stats/devices", getStats)

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

func home(c echo.Context) error {
	return c.String(http.StatusOK, "Beacon DNS API")
}
