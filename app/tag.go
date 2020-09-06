package app

import (
	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) GetTags(options *model.GetTagsOptions) (model.Tags, *model.AppError) {
	return a.Srv.Store.Tag().GetTags(options)
}

func (a *App) GetTagsCount(options *model.GetTagsOptions) (int64, *model.AppError) {
	return a.Srv.Store.Tag().GetTagsCount(options)
}

func (a *App) CreateTags(addedTags []string, time int64, teamId string, tagType string) *model.AppError {
	return a.Srv.Store.Tag().CreateTags(addedTags, time, teamId, tagType)
}

func (a *App) TopAskersForTag(interval string, teamId string, tag string) ([]*model.TopUserByTagResult, *model.AppError) {
	return a.Srv.Store.UserPointHistory().TopAskersByTag(interval, teamId, tag, 10)
}

func (a *App) TopAnswerersForTag(interval string, teamId string, tag string) ([]*model.TopUserByTagResult, *model.AppError) {
	return a.Srv.Store.UserPointHistory().TopAnswerersByTag(interval, teamId, tag, 10)
}

func (a *App) TopAnswersForTag(interval string, teamId string, tag string) ([]*model.TopPostByTagResult, *model.AppError) {
	return a.Srv.Store.UserPointHistory().TopAnswersByTag(interval, teamId, tag, 10)
}
