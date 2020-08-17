package model

import (
	"encoding/json"
	"io"
	"net/http"
	"unicode/utf8"
)

const (
	// take care of mysql's ft_min_word_len
	COLLECTION_TITLE_MIN_RUNES        = 5
	COLLECTION_TITLE_MAX_RUNES        = 1000
	COLLECTION_DESCRIPTION_MAX_LENGTH = 255
)

type Collection struct {
	Id          string `db:"Id, primarykey" json:"id"`
	CreateAt    int64  `db:"CreateAt" json:"create_at"`
	UpdateAt    int64  `db:"UpdateAt" json:"update_at"`
	DeleteAt    int64  `db:"DeleteAt" json:"delete_at"`
	TeamId      string `db:"TeamId" json:"team_id"`
	Title       string `db:"Title" json:"title"`
	Description string `db:"Description" json:"description"`
	UserId      string `db:"UserId" json:"user_id"`
}

func CollectionFromJson(data io.Reader) *Collection {
	var o *Collection
	json.NewDecoder(data).Decode(&o)
	return o
}

func (collection *Collection) ToJson() string {
	b, _ := json.Marshal(collection)
	return string(b)
}

func (o *Collection) PreSave() {
	if o.Id == "" {
		o.Id = NewId()
	}

	o.Title = SanitizeUnicode(o.Title)
	o.Description = SanitizeUnicode(o.Description)
	o.CreateAt = GetMillis()
	o.UpdateAt = o.CreateAt
}

func (o *Collection) IsValid() *AppError {
	if len(o.Id) != 26 {
		return NewAppError("Collection.IsValid", "model.collection.is_valid.id.app_error", nil, "", http.StatusBadRequest)
	}

	if o.CreateAt == 0 {
		return NewAppError("Collection.IsValid", "model.collection.is_valid.create_at.app_error", nil, "", http.StatusBadRequest)
	}

	if o.UpdateAt == 0 {
		return NewAppError("Collection.IsValid", "model.collection.is_valid.update_at.app_error", nil, "", http.StatusBadRequest)
	}

	if utf8.RuneCountInString(o.Title) > COLLECTION_TITLE_MAX_RUNES || utf8.RuneCountInString(o.Title) < COLLECTION_TITLE_MIN_RUNES {
		return NewAppError("Collection.IsValid", "model.collection.is_valid.title.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.Description) > COLLECTION_DESCRIPTION_MAX_LENGTH {
		return NewAppError("Collection.IsValid", "model.collection.is_valid.description.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.TeamId) != 26 {
		return NewAppError("Collection.IsValid", "model.collection.is_valid.team_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.UserId) != 26 {
		return NewAppError("Collection.IsValid", "model.collection.is_valid.user_id.app_error", nil, "", http.StatusBadRequest)
	}

	return nil
}
