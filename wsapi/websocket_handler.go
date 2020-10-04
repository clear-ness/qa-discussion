package wsapi

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/app"
	"github.com/clear-ness/qa-discussion/model"
)

// whという1つの関数を引数に取る
func (api *API) ApiWebSocketHandler(wh func(*model.WebSocketRequest) (map[string]interface{}, *model.AppError)) webSocketHandler {
	return webSocketHandler{api.App, wh}
}

type webSocketHandler struct {
	app         *app.App
	handlerFunc func(*model.WebSocketRequest) (map[string]interface{}, *model.AppError)
}

func NewInvalidWebSocketParamError(action string, name string) *model.AppError {
	return model.NewAppError("websocket: "+action, "api.websocket_handler.invalid_param.app_error", map[string]interface{}{"Name": name}, "", http.StatusBadRequest)
}

func (wh webSocketHandler) ServeWebSocket(conn *app.WebConn, r *model.WebSocketRequest) {
	hub := wh.app.GetHubForUserId(conn.UserId)
	if hub == nil {
		return
	}

	// webSocketリクエストを叩かれるとセッションを毎回確認する
	session, sessionErr := wh.app.GetSession(conn.GetSessionToken())
	if sessionErr != nil {
		sessionErr.DetailedError = ""
		errResp := model.NewWebSocketError(r.Seq, sessionErr)
		hub.SendMessage(conn, errResp)
		return
	}

	r.Session = *session

	var data map[string]interface{}
	var err *model.AppError

	if data, err = wh.handlerFunc(r); err != nil {
		err.DetailedError = ""
		errResp := model.NewWebSocketError(r.Seq, err)
		hub.SendMessage(conn, errResp)
		return
	}

	resp := model.NewWebSocketResponse(model.STATUS_OK, r.Seq, data)
	hub.SendMessage(conn, resp)
}
