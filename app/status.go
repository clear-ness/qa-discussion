package app

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) GetUserStatusesByIds(userIds []string) ([]*model.Status, *model.AppError) {
	statuses, err := a.Srv.Store.Status().GetByIds(userIds)
	if err != nil {
		return nil, model.NewAppError("GetUserStatusesByIds", "app.status.get.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return statuses, nil
}

func (a *App) GetStatus(userId string) (*model.Status, *model.AppError) {
	status, err := a.Srv.Store.Status().Get(userId)
	if err != nil {
		return nil, model.NewAppError("GetStatus", "app.status.get.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return status, nil
}

func (a *App) SetStatusOnline(userId string, manual bool) {
	var oldStatus string = model.STATUS_OFFLINE
	var oldTime int64
	var oldManual bool
	var status *model.Status
	var err *model.AppError

	if status, err = a.GetStatus(userId); err != nil {
		status = &model.Status{UserId: userId, Status: model.STATUS_ONLINE, Manual: false, LastActivityAt: model.GetMillis(), ActiveTeam: ""}
	} else {
		if status.Manual && !manual {
			return // manually set status always overrides non-manual one
		}

		oldStatus = status.Status
		oldTime = status.LastActivityAt
		oldManual = status.Manual

		status.Status = model.STATUS_ONLINE
		status.Manual = false // for "online" there's no manual setting
		status.LastActivityAt = model.GetMillis()
	}

	if status.Status != oldStatus || status.Manual != oldManual || status.LastActivityAt-oldTime > model.STATUS_MIN_UPDATE_TIME {
		a.Srv.Store.Status().UpdateLastActivityAt(status.UserId, status.LastActivityAt)
	}
}

func (a *App) SetStatusOffline(userId string, manual bool) {
	status, err := a.GetStatus(userId)
	if err == nil && status.Manual && !manual {
		return // manually set status always overrides non-manual one
	}

	status = &model.Status{UserId: userId, Status: model.STATUS_OFFLINE, Manual: manual, LastActivityAt: model.GetMillis(), ActiveTeam: ""}

	a.Srv.Store.Status().SaveOrUpdate(status)
}

func (a *App) SetStatusAwayIfNeeded(userId string, manual bool) {
	status, err := a.GetStatus(userId)
	if err != nil {
		status = &model.Status{UserId: userId, Status: model.STATUS_OFFLINE, Manual: manual, LastActivityAt: 0, ActiveTeam: ""}
	}

	if !manual && status.Manual {
		return // manually set status always overrides non-manual one
	}

	if !manual {
		if status.Status == model.STATUS_AWAY {
			return
		}

		if !a.IsUserAway(status.LastActivityAt) {
			return
		}
	}

	status.Status = model.STATUS_AWAY
	status.Manual = manual
	status.ActiveTeam = ""

	a.Srv.Store.Status().SaveOrUpdate(status)
}

func (a *App) IsUserAway(lastActivityAt int64) bool {
	return model.GetMillis()-lastActivityAt >= *a.Config().ServiceSettings.UserStatusAwayTimeout*1000
}
