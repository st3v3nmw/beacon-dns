package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/st3v3nmw/beacon/internal/models"
	"gorm.io/gorm"
)

// Home

func home(c echo.Context) error {
	return c.String(http.StatusOK, "Beacon API")
}

// Utils

func createHandler[T models.Creatable](c echo.Context) error {
	v := new(T)

	err := c.Bind(v)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	err = c.Validate(*v)
	if err != nil {
		return err
	}

	err = models.DB.Create(v).Error
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, v)
}

func paginate(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		q := r.URL.Query()
		page, _ := strconv.Atoi(q.Get("page"))
		if page <= 0 {
			page = 1
		}

		pageSize, _ := strconv.Atoi(q.Get("page_size"))
		switch {
		case pageSize > 100:
			pageSize = 100
		case pageSize <= 0:
			pageSize = 20
		}

		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

func getHandler[T any](c echo.Context) error {
	var vs []T
	r := c.Request()
	err := models.DB.Scopes(paginate(r)).Find(&vs).Error
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, vs)
}

// Schedules

func createSchedule(c echo.Context) error {
	return createHandler[*models.Schedule](c)
}

func getSchedules(c echo.Context) error {
	return getHandler[models.Schedule](c)
}

// Timings

func createTiming(c echo.Context) error {
	return createHandler[*models.Timing](c)
}

func getTimings(c echo.Context) error {
	return getHandler[models.Timing](c)
}

// Lists

func createList(c echo.Context) error {
	return createHandler[*models.List](c)
}

func getLists(c echo.Context) error {
	return getHandler[models.List](c)
}

// List Entries

func createListEntry(c echo.Context) error {
	return createHandler[*models.ListEntry](c)
}

func getListEntries(c echo.Context) error {
	return getHandler[models.ListEntry](c)
}
