package model

import (
	"encoding/json"
	"io"
	"net/http"
)

const (
	GROUP_TYPE_PUBLIC  = "public"
	GROUP_TYPE_PRIVATE = "private"

	GROUP_DESCRIPTION_MAX_LENGTH = 255
	GROUP_NAME_MIN_LENGTH        = 3
	GROUP_SEARCH_DEFAULT_LIMIT   = 10
)

type UserGroup struct {
	Id          string `db:"Id, primarykey" json:"id"`
	Type        string `db:"Type" json:"type"`
	CreateAt    int64  `db:"CreateAt" json:"create_at"`
	UpdateAt    int64  `db:"UpdateAt" json:"update_at"`
	DeleteAt    int64  `db:"DeleteAt" json:"delete_at"`
	TeamId      string `db:"TeamId" json:"team_id"`
	Name        string `db:"Name" json:"name"`
	Description string `db:"Description" json:"description"`
	UserId      string `db:"UserId" json:"user_id"`
}

func UserGroupFromJson(data io.Reader) *UserGroup {
	var o *UserGroup
	json.NewDecoder(data).Decode(&o)
	return o
}

func (o *UserGroup) PreSave() {
	if o.Id == "" {
		o.Id = NewId()
	}

	if o.Type == "" {
		o.Type = GROUP_TYPE_PRIVATE
	}

	o.Name = SanitizeUnicode(o.Name)
	o.Description = SanitizeUnicode(o.Description)
	o.CreateAt = GetMillis()
	o.UpdateAt = o.CreateAt
}

func (o *UserGroup) IsValid() *AppError {
	if len(o.Id) != 26 {
		return NewAppError("UserGroup.IsValid", "model.group.is_valid.id.app_error", nil, "", http.StatusBadRequest)
	}

	if o.Type != GROUP_TYPE_PUBLIC && o.Type != GROUP_TYPE_PRIVATE {
		return NewAppError("UserGroup.IsValid", "model.group.is_valid.type.app_error", nil, "", http.StatusBadRequest)
	}

	if o.CreateAt == 0 {
		return NewAppError("UserGroup.IsValid", "model.group.is_valid.create_at.app_error", nil, "", http.StatusBadRequest)
	}

	if o.UpdateAt == 0 {
		return NewAppError("UserGroup.IsValid", "model.group.is_valid.update_at.app_error", nil, "", http.StatusBadRequest)
	}

	if !IsValidUserGroupIdentifier(o.Name) {
		return NewAppError("UserGroup.IsValid", "model.group.is_valid.name.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.Description) > GROUP_DESCRIPTION_MAX_LENGTH {
		return NewAppError("UserGroup.IsValid", "model.group.is_valid.description.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.TeamId) != 26 {
		return NewAppError("UserGroup.IsValid", "model.group.is_valid.team_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.UserId) != 26 {
		return NewAppError("UserGroup.IsValid", "model.group.is_valid.user_id.app_error", nil, "", http.StatusBadRequest)
	}

	return nil
}

func (group *UserGroup) ToJson() string {
	b, _ := json.Marshal(group)
	return string(b)
}

func (o *UserGroup) DeepCopy() *UserGroup {
	copy := *o
	return &copy
}

func (o *UserGroup) PreUpdate() {
	o.UpdateAt = GetMillis()
	o.Name = SanitizeUnicode(o.Name)
	o.Description = SanitizeUnicode(o.Description)
}
