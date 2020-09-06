package model

import (
	"encoding/json"
	"net/http"
)

const (
	ACCESS_TOKEN_GRANT_TYPE  = "authorization_code"
	REFRESH_TOKEN_GRANT_TYPE = "refresh_token"
	ACCESS_TOKEN_TYPE        = "bearer"
)

type AccessData struct {
	ClientId     string `db:"ClientId" json:"client_id"`
	UserId       string `db:"UserId" json:"user_id"`
	Token        string `db:"Token, primarykey" json:"token"`
	RefreshToken string `db:"RefreshToken" json:"refresh_token"`
	RedirectUri  string `db:"RedirectUri" json:"redirect_uri"`
	ExpiresAt    int64  `db:"ExpiresAt" json:"expires_at"`
	Scope        string `db:"Scope" json:"scope"`
}

func (ad *AccessData) IsValid() *AppError {
	if len(ad.ClientId) != 26 {
		return NewAppError("AccessData.IsValid", "model.access.is_valid.client_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(ad.UserId) != 26 {
		return NewAppError("AccessData.IsValid", "model.access.is_valid.user_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(ad.Token) != 26 {
		return NewAppError("AccessData.IsValid", "model.access.is_valid.access_token.app_error", nil, "", http.StatusBadRequest)
	}

	if len(ad.RefreshToken) > 26 {
		return NewAppError("AccessData.IsValid", "model.access.is_valid.refresh_token.app_error", nil, "", http.StatusBadRequest)
	}

	if len(ad.RedirectUri) == 0 || len(ad.RedirectUri) > 256 || !IsValidHttpUrl(ad.RedirectUri) {
		return NewAppError("AccessData.IsValid", "model.access.is_valid.redirect_uri.app_error", nil, "", http.StatusBadRequest)
	}

	return nil
}

func (ad *AccessData) IsExpired() bool {
	if ad.ExpiresAt <= 0 {
		return false
	}

	if GetMillis() > ad.ExpiresAt {
		return true
	}

	return false
}

type AccessResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int32  `json:"expires_in"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}

func (ar *AccessResponse) ToJson() string {
	b, _ := json.Marshal(ar)
	return string(b)
}
