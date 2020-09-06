package model

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"unicode/utf8"
)

type OAuthApp struct {
	Id           string      `db:"Id, primarykey" json:"id"`
	UserId       string      `db:"UserId" json:"user_id"`
	ClientSecret string      `db:"ClientSecret" json:"client_secret"`
	Name         string      `db:"Name" json:"name"`
	Description  string      `db:"Description" json:"description"`
	IconURL      string      `db:"IconURL" json:"icon_url"`
	URLs         StringArray `db:"URLs" json:"urls"`
	Homepage     string      `db:"Homepage" json:"homepage"`
	CreateAt     int64       `db:"CreateAt" json:"create_at"`
	UpdateAt     int64       `db:"UpdateAt" json:"update_at"`
}

func (a *OAuthApp) IsValid() *AppError {
	if len(a.Id) != 26 {
		return NewAppError("OAuthApp.IsValid", "model.oauth.is_valid.app_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(a.UserId) != 26 {
		return NewAppError("OAuthApp.IsValid", "model.oauth.is_valid.user_id.app_error", nil, "app_id="+a.Id, http.StatusBadRequest)
	}

	if len(a.ClientSecret) == 0 || len(a.ClientSecret) > 128 {
		return NewAppError("OAuthApp.IsValid", "model.oauth.is_valid.client_secret.app_error", nil, "app_id="+a.Id, http.StatusBadRequest)
	}

	if len(a.Name) == 0 || len(a.Name) > 64 {
		return NewAppError("OAuthApp.IsValid", "model.oauth.is_valid.name.app_error", nil, "app_id="+a.Id, http.StatusBadRequest)
	}

	if len(a.URLs) == 0 || len(fmt.Sprintf("%s", a.URLs)) > 1024 {
		return NewAppError("OAuthApp.IsValid", "model.oauth.is_valid.url.app_error", nil, "app_id="+a.Id, http.StatusBadRequest)
	}

	for _, url := range a.URLs {
		if !IsValidHttpUrl(url) {
			return NewAppError("OAuthApp.IsValid", "model.oauth.is_valid.url.app_error", nil, "", http.StatusBadRequest)
		}
	}

	if len(a.Homepage) == 0 || len(a.Homepage) > 256 || !IsValidHttpUrl(a.Homepage) {
		return NewAppError("OAuthApp.IsValid", "model.oauth.is_valid.homepage.app_error", nil, "app_id="+a.Id, http.StatusBadRequest)
	}

	if utf8.RuneCountInString(a.Description) > 512 {
		return NewAppError("OAuthApp.IsValid", "model.oauth.is_valid.description.app_error", nil, "app_id="+a.Id, http.StatusBadRequest)
	}

	if len(a.IconURL) > 0 {
		if len(a.IconURL) > 512 || !IsValidHttpUrl(a.IconURL) {
			return NewAppError("OAuthApp.IsValid", "model.oauth.is_valid.icon_url.app_error", nil, "app_id="+a.Id, http.StatusBadRequest)
		}
	}

	return nil
}

func OAuthAppFromJson(data io.Reader) *OAuthApp {
	var app *OAuthApp
	json.NewDecoder(data).Decode(&app)
	return app
}

func (a *OAuthApp) ToJson() string {
	b, _ := json.Marshal(a)
	return string(b)
}

func (a *OAuthApp) PreSave() {
	if a.Id == "" {
		a.Id = NewId()
	}

	if a.ClientSecret == "" {
		a.ClientSecret = NewId()
	}

	a.CreateAt = GetMillis()
	a.UpdateAt = a.CreateAt
}

func (a *OAuthApp) PreUpdate() {
	a.UpdateAt = GetMillis()
}

func OAuthAppListToJson(list []*OAuthApp) string {
	b, _ := json.Marshal(list)
	return string(b)
}

func (a *OAuthApp) Sanitize() {
	a.ClientSecret = ""
}

func (a *OAuthApp) IsValidRedirectURL(url string) bool {
	for _, u := range a.URLs {
		if u == url {
			return true
		}
	}

	return false
}
