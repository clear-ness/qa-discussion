package model

import (
	"net/http"
	"unicode/utf8"
)

const (
	// take care of mysql's ft_min_word_len
	TAG_MIN_RUNES = 3
	TAG_MAX_RUNES = 64

	TAG_TYPE_REVIEW = "review"
	TAG_TYPE_SYSTEM = "system"

	AUTOCOMPLETE_TAGS_LIMIT = 20

	// TODO: more system tags
	SYSTEM_TAG_FIRST_POSTS  = "first_posts"
	SYSTEM_TAG_LATE_ANSWERS = "late_answers"

	LATE_ANSWERS_MILLIS = int64(365 * 60 * 60 * 24 * 1000)

	REVIEW_TAG_ABUSE           = "spam"
	REVIEW_TAG_SPAM            = "abuse"
	REVIEW_TAG_DUPLICATE       = "duplicate"
	REVIEW_TAG_INVALID_CONTENT = "invalid_content"
	REVIEW_TAG_LOW_QUALITY     = "low_quality"
)

type Tag struct {
	Content   string `db:"Content, primarykey" json:"content"`
	TeamId    string `db:"TeamId, primarykey" json:"team_id"`
	Type      string `db:"Type, primarykey" json:"type"`
	PostCount int    `db:"PostCount" json:"post_count"`
	CreateAt  int64  `db:"CreateAt" json:"create_at"`
	UpdateAt  int64  `db:"UpdateAt" json:"update_at"`
}

func (o *Tag) IsValid() *AppError {
	if utf8.RuneCountInString(o.Content) > TAG_MAX_RUNES || utf8.RuneCountInString(o.Content) < TAG_MIN_RUNES {
		return NewAppError("Tag.IsValid", "model.tag.is_valid.content.app_error", nil, "", http.StatusBadRequest)
	}

	if o.TeamId != "" && len(o.TeamId) != 26 {
		return NewAppError("Tag.IsValid", "model.tag.is_valid.team_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.Type) > 0 && o.Type != TAG_TYPE_REVIEW && o.Type != TAG_TYPE_SYSTEM {
		return NewAppError("Tag.IsValid", "model.tag.is_valid.type.app_error", nil, "", http.StatusBadRequest)
	}

	if o.CreateAt == 0 {
		return NewAppError("Tag.IsValid", "model.tag.is_valid.create_at.app_error", nil, "", http.StatusBadRequest)
	}

	if o.UpdateAt == 0 {
		return NewAppError("Tag.IsValid", "model.tag.is_valid.update_at.app_error", nil, "", http.StatusBadRequest)
	}

	return nil
}

func (o *Tag) PreSave() {
	if o.CreateAt == 0 {
		o.CreateAt = GetMillis()
	}

	o.UpdateAt = o.CreateAt
}

type GetTagsOptions struct {
	FromDate int64
	ToDate   int64
	SortType string
	Min      *int
	Max      *int
	InName   string
	Content  string
	Page     int
	PerPage  int
	TeamId   string
	Type     string
}
