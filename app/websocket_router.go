package app

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
)

type webSocketHandler interface {
	ServeWebSocket(*WebConn, *model.WebSocketRequest)
}

type WebSocketRouter struct {
	server   *Server
	app      *App
	handlers map[string]webSocketHandler
}

// wsapiから呼ばれる
func (wr *WebSocketRouter) Handle(action string, handler webSocketHandler) {
	wr.handlers[action] = handler
}

// httpからのupgradeを前提としたwebSocket、を仕様とする。
func (wr *WebSocketRouter) ServeWebSocket(conn *WebConn, r *model.WebSocketRequest) {
	if r.Action == "" {
		err := model.NewAppError("ServeWebSocket", "api.web_socket_router.no_action.app_error", nil, "", http.StatusBadRequest)
		returnWebSocketError(wr.app, conn, r, err)
		return
	}

	if r.Seq <= 0 {
		err := model.NewAppError("ServeWebSocket", "api.web_socket_router.bad_seq.app_error", nil, "", http.StatusBadRequest)
		returnWebSocketError(wr.app, conn, r, err)
		return
	}

	if !conn.IsAuthenticated() {
		err := model.NewAppError("ServeWebSocket", "api.web_socket_router.not_authenticated.app_error", nil, "", http.StatusUnauthorized)
		returnWebSocketError(wr.app, conn, r, err)
		return
	}

	// 各actionに応じたハンドラを呼ぶ
	handler, ok := wr.handlers[r.Action]
	if !ok {
		err := model.NewAppError("ServeWebSocket", "api.web_socket_router.bad_action.app_error", nil, "", http.StatusInternalServerError)
		returnWebSocketError(wr.app, conn, r, err)
		return
	}

	handler.ServeWebSocket(conn, r)
}

func returnWebSocketError(app *App, conn *WebConn, r *model.WebSocketRequest, err *model.AppError) {
	hub := app.GetHubForUserId(conn.UserId)
	if hub == nil {
		return
	}

	err.DetailedError = ""
	errorResp := model.NewWebSocketError(r.Seq, err)

	hub.SendMessage(conn, errorResp)
}
