package wsapi

import (
	"github.com/clear-ness/qa-discussion/model"
)

func (api *API) InitUser() {
	// app/websocket_routerのHandleを呼んでいる
	api.Router.Handle("user_update_active_status", api.ApiWebSocketHandler(api.userUpdateActiveStatus))
}

func (api *API) userUpdateActiveStatus(req *model.WebSocketRequest) (map[string]interface{}, *model.AppError) {
	var ok bool
	var userIsActive bool
	if userIsActive, ok = req.Data["user_is_active"].(bool); !ok {
		return nil, NewInvalidWebSocketParamError(req.Action, "user_is_active")
	}

	var manual bool
	if manual, ok = req.Data["manual"].(bool); !ok {
		manual = false
	}

	if userIsActive {
		api.App.SetStatusOnline(req.Session.UserId, manual)
	} else {
		api.App.SetStatusAwayIfNeeded(req.Session.UserId, manual)
	}

	return nil, nil
}
