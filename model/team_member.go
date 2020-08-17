package model

import (
	"encoding/json"
	"net/http"
)

const (
	TEAM_MEMBER_TYPE_NORMAL = "normal"
	TEAM_MEMBER_TYPE_ADMIN  = "admin"

	TEAM_MEMBER_SORT_TYPE_USERNAME = "username"
)

type TeamMember struct {
	TeamId   string `db:"TeamId" json:"team_id"`
	UserId   string `db:"UserId" json:"user_id"`
	Type     string `db:"Type" json:"type"`
	Points   int    `db:"Points" json:"points,omitempty"`
	DeleteAt int64  `db:"DeleteAt" json:"delete_at"`
}

func (o *TeamMember) IsValid() *AppError {
	if len(o.TeamId) != 26 {
		return NewAppError("TeamMember.IsValid", "model.team_member.is_valid.team_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.UserId) != 26 {
		return NewAppError("TeamMember.IsValid", "model.team_member.is_valid.user_id.app_error", nil, "", http.StatusBadRequest)
	}

	if o.Type != TEAM_MEMBER_TYPE_NORMAL && o.Type != TEAM_MEMBER_TYPE_ADMIN {
		return NewAppError("TeamMember.IsValid", "model.team_member.is_valid.type.app_error", nil, "", http.StatusBadRequest)
	}

	return nil
}

func (o *TeamMember) ToJson() string {
	b, _ := json.Marshal(o)
	return string(b)
}

type TeamMembersGetOptions struct {
	// Sort the team members. Accepts "Username", but defaults to "Id".
	Sort string
	// If true, exclude team members whose corresponding user is deleted.
	ExcludeDeletedUsers bool
	// member type
	Type string
}

func TeamMembersToJson(o []*TeamMember) string {
	if b, err := json.Marshal(o); err != nil {
		return "[]"
	} else {
		return string(b)
	}
}
