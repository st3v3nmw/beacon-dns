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

type customValidator struct {
	validator *validator.Validate
}

func (cv *customValidator) Validate(i interface{}) error {
	err := cv.validator.Struct(i)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func New(addr string) {
	e := echo.New()

	e.Validator = &customValidator{validator: validator.New()}

	e.GET("/", home)

	e.GET("/:filter/dns-query", queryDNS)
	e.POST("/:filter/dns-query", queryDNS)

	e.POST("/lists", createList)
	e.GET("/lists", getLists)

	e.POST("/list-entries", createListEntry)
	e.GET("/list-entries", getListEntries)

	API = &APIService{
		address: addr,
		echo:    e,
	}
}

func Start() error {
	return API.echo.Start(API.address)
}
