package api

import (
	"net/http"

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

	ch := querylog.Broadcaster.Subscribe()
	defer querylog.Broadcaster.Unsubscribe(ch)
	for query := range ch {
		err := ws.WriteJSON(query)
		if err != nil {
			return err
		}
	}

	return nil
}
