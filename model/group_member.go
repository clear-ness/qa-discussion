package model

import (
	"encoding/json"
	"net/http"
)

const (
	GROUP_MEMBER_TYPE_NORMAL = "normal"
	GROUP_MEMBER_TYPE_ADMIN  = "admin"

	GROUP_MEMBER_SEARCH_DEFAULT_LIMIT = 10
)

type GroupMember struct {
	GroupId string `db:"GroupId" json:"group_id"`
	UserId  string `db:"UserId" json:"user_id"`
	Type    string `db:"Type" json:"type"`
}

func (o *GroupMember) IsValid() *AppError {
	if len(o.GroupId) != 26 {
		return NewAppError("GroupMember.IsValid", "model.group_member.is_valid.group_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.UserId) != 26 {
		return NewAppError("GroupMember.IsValid", "model.group_member.is_valid.user_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.Type) > 0 && o.Type != GROUP_MEMBER_TYPE_NORMAL && o.Type != GROUP_MEMBER_TYPE_ADMIN {
		return NewAppError("GroupMember.IsValid", "model.group_member.is_valid.type.app_error", nil, "", http.StatusBadRequest)
	}

	return nil
}

func (o *GroupMember) ToJson() string {
	b, _ := json.Marshal(o)
	return string(b)
}

type GroupMembers []GroupMember

func (o *GroupMembers) ToJson() string {
	if b, err := json.Marshal(o); err != nil {
		return "[]"
	} else {
		return string(b)
	}
}
