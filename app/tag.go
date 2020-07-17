package app

import (
	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) GetTags(options *model.GetTagsOptions) (model.Tags, *model.AppError) {
	return a.Srv.Store.Tag().GetTags(options)
}
