package model

import (
	"encoding/json"
	"io"
	"net/http"
)

const (
	AUTHCODE_RESPONSE_TYPE = "code"
	IMPLICIT_RESPONSE_TYPE = "token"

	DEFAULT_SCOPE = "all"

	AUTHCODE_EXPIRE_TIME = 60 * 10 // 10 minutes
)

type AuthData struct {
	ClientId    string `db:"ClientId" json:"client_id"`
	UserId      string `db:"UserId" json:"user_id"`
	Code        string `db:"Code, primarykey" json:"code"`
	ExpiresIn   int32  `db:"ExpiresIn" json:"expires_in"`
	CreateAt    int64  `db:"CreateAt" json:"create_at"`
	RedirectUri string `db:"RedirectUri" json:"redirect_uri"`
	State       string `db:"State" json:"state"`
	Scope       string `db:"Scope" json:"scope"`
}

func (ad *AuthData) PreSave() {
	if ad.ExpiresIn == 0 {
		ad.ExpiresIn = AUTHCODE_EXPIRE_TIME
	}

	if ad.CreateAt == 0 {
		ad.CreateAt = GetMillis()
	}

	if len(ad.Scope) == 0 {
		ad.Scope = DEFAULT_SCOPE
	}
}

func (ad *AuthData) IsValid() *AppError {
	if len(ad.ClientId) != 26 {
		return NewAppError("AuthData.IsValid", "model.authorize.is_valid.client_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(ad.UserId) != 26 {
		return NewAppError("AuthData.IsValid", "model.authorize.is_valid.user_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(ad.Code) == 0 || len(ad.Code) > 128 {
		return NewAppError("AuthData.IsValid", "model.authorize.is_valid.auth_code.app_error", nil, "client_id="+ad.ClientId, http.StatusBadRequest)
	}

	if ad.ExpiresIn == 0 {
		return NewAppError("AuthData.IsValid", "model.authorize.is_valid.expires.app_error", nil, "", http.StatusBadRequest)
	}

	if len(ad.RedirectUri) > 256 || !IsValidHttpUrl(ad.RedirectUri) {
		return NewAppError("AuthData.IsValid", "model.authorize.is_valid.redirect_uri.app_error", nil, "client_id="+ad.ClientId, http.StatusBadRequest)
	}

	if len(ad.State) > 1024 {
		return NewAppError("AuthData.IsValid", "model.authorize.is_valid.state.app_error", nil, "client_id="+ad.ClientId, http.StatusBadRequest)
	}

	if len(ad.Scope) > 128 {
		return NewAppError("AuthData.IsValid", "model.authorize.is_valid.scope.app_error", nil, "client_id="+ad.ClientId, http.StatusBadRequest)
	}

	return nil
}

func (ad *AuthData) IsExpired() bool {
	return GetMillis() > ad.CreateAt+int64(ad.ExpiresIn*1000)
}

type AuthorizeRequest struct {
	ResponseType string `json:"response_type"`
	ClientId     string `json:"client_id"`
	RedirectUri  string `json:"redirect_uri"`
	Scope        string `json:"scope"`
	State        string `json:"state"`
}

func AuthorizeRequestFromJson(data io.Reader) *AuthorizeRequest {
	var ar *AuthorizeRequest
	json.NewDecoder(data).Decode(&ar)
	return ar
}

func (ar *AuthorizeRequest) IsValid() *AppError {
	if len(ar.ResponseType) == 0 {
		return NewAppError("AuthData.IsValid", "model.authorize.is_valid.response_type.app_error", nil, "", http.StatusBadRequest)
	}

	if len(ar.ClientId) != 26 {
		return NewAppError("AuthData.IsValid", "model.authorize.is_valid.client_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(ar.RedirectUri) == 0 || len(ar.RedirectUri) > 256 || !IsValidHttpUrl(ar.RedirectUri) {
		return NewAppError("AuthData.IsValid", "model.authorize.is_valid.redirect_uri.app_error", nil, "client_id="+ar.ClientId, http.StatusBadRequest)
	}

	if len(ar.Scope) > 128 {
		return NewAppError("AuthData.IsValid", "model.authorize.is_valid.scope.app_error", nil, "client_id="+ar.ClientId, http.StatusBadRequest)
	}

	if len(ar.State) > 1024 {
		return NewAppError("AuthData.IsValid", "model.authorize.is_valid.state.app_error", nil, "client_id="+ar.ClientId, http.StatusBadRequest)
	}

	return nil
}
