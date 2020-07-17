package app

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) GetNotificationSettingForUser(userId string) (*model.NotificationSetting, *model.AppError) {
	res, err := a.Srv.Store.NotificationSetting().Get(userId)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (a *App) UpdateNotificationSettingForUser(userId string, inboxInterval string) *model.AppError {
	if err := a.Srv.Store.NotificationSetting().Save(userId, inboxInterval); err != nil {
		err.StatusCode = http.StatusBadRequest
		return err
	}

	return nil
}
