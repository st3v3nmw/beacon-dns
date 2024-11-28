package api

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/st3v3nmw/beacon/internal/querylog"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: checkOrigin,
	}
)

func checkOrigin(r *http.Request) bool {
	return true
}

func watch(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	defer ws.Close()

	var clients []string
	clientsParam := c.QueryParam("clients")
	if clientsParam != "" {
		clients = strings.Split(clientsParam, ",")
	}

	ch := querylog.Broadcaster.Subscribe()
	defer querylog.Broadcaster.Unsubscribe(ch)
	for query := range ch {
		if len(clients) == 0 || slices.Contains(clients, query.Hostname) {
			err := ws.WriteJSON(query)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
