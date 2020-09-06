package app

import (
	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) GetWebhooksHistoriesPage(teamId string, page, perPage int) ([]*model.WebhooksHistory, *model.AppError) {
	return a.Srv.Store.WebhooksHistory().GetWebhooksHistoriesPage(teamId, page*perPage, perPage)
}
