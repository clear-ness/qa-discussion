package api

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
)

func (api *API) InitNotificationSetting() {
	api.BaseRoutes.NotificationSettingForUser.Handle("", api.ApiSessionRequired(getNotificationSettingForUser)).Methods("GET")

	api.BaseRoutes.NotificationSettingForUser.Handle("", api.ApiSessionRequired(updateNotificationSettingForUser)).Methods("PUT")
}

func getNotificationSettingForUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToUser(c.App.Session, c.Params.UserId) {
		c.SetPermissionError(model.PERMISSION_EDIT_OTHER_USERS)
		return
	}

	setting, err := c.App.GetNotificationSettingForUser(c.Params.UserId)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(setting.ToJson()))
}

func updateNotificationSettingForUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToUser(c.App.Session, c.Params.UserId) {
		c.SetPermissionError(model.PERMISSION_EDIT_OTHER_USERS)
		return
	}

	// if interval of params is "", NotificationSettings's interval value will be "".
	if err := c.App.UpdateNotificationSettingForUser(c.Params.UserId, c.Params.InboxInterval); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}
