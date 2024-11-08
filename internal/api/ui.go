package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// home page
func home(c echo.Context) error {
	return c.Render(http.StatusOK, "home.html", map[string]interface{}{})
}
