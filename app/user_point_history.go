package app

import (
	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) GetUserPointHistoryForUser(toDate int64, userId string, page, perPage int, teamId string) ([]*model.UserPointHistory, *model.AppError) {
	return a.Srv.Store.UserPointHistory().GetUserPointHistoryBeforeTime(toDate, userId, page, perPage, teamId)
}
