package model

import (
	"net/http"
)

type OAuthAuthorizedApp struct {
	UserId   string `db:"UserId" json:"user_id"`
	ClientId string `db:"ClientId" json:"client_id"`
	Scope    string `db:"Scope" json:"scope"`
}

type OAuthAuthorizedApps []OAuthAuthorizedApp

func (o *OAuthAuthorizedApp) IsValid() *AppError {
	if len(o.UserId) != 26 {
		return NewAppError("OAuthAuthorizedApp.IsValid", "model.oauth_authorized_app.is_valid.user_id.app_error", nil, "user_id="+o.UserId, http.StatusBadRequest)
	}

	if len(o.ClientId) != 26 {
		return NewAppError("OAuthAuthorizedApp.IsValid", "model.oauth_authorized_app.is_valid.client_id.app_error", nil, "client_id="+o.ClientId, http.StatusBadRequest)
	}

	if len(o.Scope) > 128 {
		return NewAppError("OAuthAuthorizedApp.IsValid", "model.oauth_authorized_app.is_valid.scope.app_error", nil, "scope="+o.Scope, http.StatusBadRequest)
	}

	return nil
}
