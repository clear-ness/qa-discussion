package app

import (
	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) GetUserPointHistoryForUser(toDate int64, userId string, page, perPage int) ([]*model.UserPointHistory, *model.AppError) {
	return a.Srv.Store.UserPointHistory().GetUserPointHistoryBeforeTime(toDate, userId, page, perPage)
}
