package app

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) MakePermissionError(permission *model.Permission) *model.AppError {
	return model.NewAppError("Permissions", "api.context.permissions.app_error", nil, "userId="+a.Session.UserId+", "+"permission="+permission.Id, http.StatusForbidden)
}

func (a *App) SessionHasPermissionTo(session model.Session, permission *model.Permission) bool {
	if session.UserId == "" {
		return false
	}

	user, err := a.GetUser(session.UserId)
	if err != nil || user == nil {
		return false
	}

	var permissions []string
	switch user.Type {
	case model.USER_TYPE_NORMAL:
		permissions = model.ROLE_NORMAL.Permissions
	case model.USER_TYPE_MODERATOR:
		permissions = model.ROLE_MODERATOR.Permissions
	case model.USER_TYPE_ADMIN:
		permissions = model.ROLE_ADMIN.Permissions
	default:
		return false
	}

	for _, allowedPermission := range permissions {
		if allowedPermission == permission.Id {
			return true
		}
	}

	return false
}

func (a *App) SessionHasPermissionToUser(session model.Session, userId string) bool {
	if userId == "" {
		return false
	}

	if session.UserId == userId {
		return true
	}

	if a.SessionHasPermissionTo(session, model.PERMISSION_EDIT_OTHER_USERS) {
		return true
	}

	return false
}

// チームはシステム全体とは影響を別にしたい
// ので、権限の検証はチーム内でしかしない。
// TODO: system adminは別？
func (a *App) SessionHasPermissionToTeam(session model.Session, teamId string, permission *model.Permission) bool {
	if teamId == "" {
		return false
	}

	teamMember := session.GetTeamByTeamId(teamId)
	if teamMember != nil {
		if a.TeamMemberHasPermissionTo(teamMember.Type, permission) {
			return true
		}
	}

	return false
}

func (a *App) TeamMemberHasPermissionTo(memberType string, permission *model.Permission) bool {
	var permissions []string
	switch memberType {
	case model.TEAM_MEMBER_TYPE_NORMAL:
		permissions = model.ROLE_TEAM_MEMBER_TYPE_NORMAL.Permissions
	case model.TEAM_MEMBER_TYPE_ADMIN:
		permissions = model.ROLE_TEAM_MEMBER_TYPE_ADMIN.Permissions
	default:
		return false
	}

	for _, allowedPermission := range permissions {
		if allowedPermission == permission.Id {
			return true
		}
	}

	return false
}

// グループはチームに依存する概念のため、
// グループ → チーム の順に権限を検証する
func (a *App) SessionHasPermissionToGroup(session model.Session, groupId string, permission *model.Permission) bool {
	if groupId == "" {
		return false
	}

	memberTypes, err := a.Srv.Store.UserGroup().GetAllGroupMembersForUser(session.UserId)
	if err == nil {
		if memberType, ok := memberTypes[groupId]; ok {
			if a.GroupMemberHasPermissionTo(memberType, permission) {
				return true
			}
		}
	}

	group, err := a.GetGroup(groupId)
	if err == nil && group.TeamId != "" {
		return a.SessionHasPermissionToTeam(session, group.TeamId, permission)
	}

	return false
}

func (a *App) GroupMemberHasPermissionTo(memberType string, permission *model.Permission) bool {
	var permissions []string
	switch memberType {
	case model.GROUP_MEMBER_TYPE_NORMAL:
		permissions = model.ROLE_GROUP_MEMBER_TYPE_NORMAL.Permissions
	case model.GROUP_MEMBER_TYPE_ADMIN:
		permissions = model.ROLE_GROUP_MEMBER_TYPE_ADMIN.Permissions
	default:
		return false
	}

	for _, allowedPermission := range permissions {
		if allowedPermission == permission.Id {
			return true
		}
	}

	return false
}
