package api

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"

	"github.com/gorilla/websocket"
)

func (api *API) InitWebSocket() {
	// http (cookie)からwebSocketへのupgrade
	api.BaseRoutes.ApiRoot.Handle("/{websocket:websocket(?:\\/)?}", api.ApiSessionRequiredTrustRequester(connectWebSocket)).Methods("GET")
}

func connectWebSocket(c *Context, w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  model.SOCKET_MAX_MESSAGE_SIZE_KB,
		WriteBufferSize: model.SOCKET_MAX_MESSAGE_SIZE_KB,
		CheckOrigin:     c.App.OriginChecker(),
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		c.Err = model.NewAppError("connect", "api.web_socket.connect.upgrade.app_error", nil, "", http.StatusInternalServerError)
		return
	}

	wc := c.App.NewWebConn(ws, c.App.Session)

	if len(c.App.Session.UserId) > 0 {
		c.App.HubRegister(wc)
	}

	wc.Pump()
}
