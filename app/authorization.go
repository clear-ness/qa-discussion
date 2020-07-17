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
