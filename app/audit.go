package app

import (
	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) GetAudits(userId string, limit int) (model.Audits, *model.AppError) {
	return a.Srv.Store.Audit().Get(userId, 0, limit)
}

func (a *App) GetAuditsPage(userId string, page int, perPage int) (model.Audits, *model.AppError) {
	return a.Srv.Store.Audit().Get(userId, page*perPage, perPage)
}
