package api

import (
	"html/template"
	"io"
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

	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("web/template/*.html")),
	}
	e.Renderer = renderer

	e.GET("/", home)

	e.GET("/:filter/dns-query", queryDNS)
	e.POST("/:filter/dns-query", queryDNS)

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
	err := cv.validator.Struct(i)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

// template renderer
type TemplateRenderer struct {
	templates *template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}
