package app

import (
	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) TagsMigration() {
	options := &model.GetTagsOptions{TeamId: "", Type: model.TAG_TYPE_SYSTEM}

	var count int64
	var err *model.AppError
	if count, err = a.Srv.Store.Tag().GetTagsCount(options); err != nil {
		return
	}

	curTime := model.GetMillis()

	if count <= 0 {
		systemTags := []string{model.SYSTEM_TAG_FIRST_POSTS, model.SYSTEM_TAG_LATE_ANSWERS}
		if err := a.Srv.Store.Tag().CreateTags(systemTags, curTime, "", model.TAG_TYPE_SYSTEM); err != nil {
			return
		}
	}

	options = &model.GetTagsOptions{TeamId: "", Type: model.TAG_TYPE_REVIEW}
	if count, err = a.Srv.Store.Tag().GetTagsCount(options); err != nil {
		return
	}

	if count <= 0 {
		reviewTags := []string{model.REVIEW_TAG_ABUSE, model.REVIEW_TAG_SPAM, model.REVIEW_TAG_DUPLICATE, model.REVIEW_TAG_INVALID_CONTENT, model.REVIEW_TAG_LOW_QUALITY}
		if err := a.Srv.Store.Tag().CreateTags(reviewTags, curTime, "", model.TAG_TYPE_REVIEW); err != nil {
			return
		}
	}
}

func (a *App) InitMigrations() {
	a.TagsMigration()
}
