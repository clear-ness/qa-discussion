package model

import (
	"net/http"
	"unicode/utf8"
)

const (
	// take care of mysql's ft_min_word_len
	TAG_MIN_RUNES = 3
	TAG_MAX_RUNES = 64

	AUTOCOMPLETE_TAGS_LIMIT = 20
)

type Tag struct {
	Content   string `db:"Content, primarykey" json:"content"`
	TeamId    string `db:"TeamId, primarykey" json:"team_id"`
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
	Page     int
	PerPage  int
	TeamId   string
}
