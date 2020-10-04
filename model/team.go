package model

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const (
	TEAM_TYPE_PUBLIC  = "public"
	TEAM_TYPE_PRIVATE = "private"

	TEAM_EMAIL_MAX_LENGTH           = 128
	TEAM_DESCRIPTION_MAX_LENGTH     = 255
	TEAM_NAME_MAX_LENGTH            = 64
	TEAM_NAME_MIN_LENGTH            = 3
	TEAM_ALLOWED_DOMAINS_MAX_LENGTH = 500

	TEAM_SEARCH_DEFAULT_LIMIT = 10
)

type Team struct {
	Id                string `db:"Id, primarykey" json:"id"`
	Type              string `db:"Type" json:"type"`
	Name              string `db:"Name" json:"name"`
	Description       string `db:"Description" json:"description"`
	Email             string `db:"Email" json:"email"`
	AllowedDomains    string `db:"AllowedDomains" json:"allowed_domains"`
	InviteId          string `db:"InviteId" json:"invite_id"`
	CreateAt          int64  `db:"CreateAt" json:"create_at"`
	UpdateAt          int64  `db:"UpdateAt" json:"update_at"`
	DeleteAt          int64  `db:"DeleteAt" json:"delete_at"`
	LastPictureUpdate int64  `db:"LastPictureUpdate" json:"last_picture_update,omitempty"`

	TeamImageLink string `db:"-" json:"team_image_link,omitempty`
}

func (o *Team) IsValid() *AppError {
	if len(o.Id) != 26 {
		return NewAppError("Team.IsValid", "model.team.is_valid.id.app_error", nil, "", http.StatusBadRequest)
	}

	if o.Type != TEAM_TYPE_PUBLIC && o.Type != TEAM_TYPE_PRIVATE {
		return NewAppError("Team.IsValid", "model.team.is_valid.type.app_error", nil, "", http.StatusBadRequest)
	}

	if o.CreateAt == 0 {
		return NewAppError("Team.IsValid", "model.team.is_valid.create_at.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if o.UpdateAt == 0 {
		return NewAppError("Team.IsValid", "model.team.is_valid.update_at.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if len(o.Email) > TEAM_EMAIL_MAX_LENGTH || len(o.Email) == 0 || !IsValidEmail(o.Email) {
		return NewAppError("Team.IsValid", "model.team.is_valid.email.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if len(o.Description) > TEAM_DESCRIPTION_MAX_LENGTH {
		return NewAppError("Team.IsValid", "model.team.is_valid.description.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if len(o.InviteId) == 0 {
		return NewAppError("Team.IsValid", "model.team.is_valid.invite_id.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if len(o.Name) > TEAM_NAME_MAX_LENGTH {
		return NewAppError("Team.IsValid", "model.team.is_valid.url.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if IsReservedTeamName(o.Name) {
		return NewAppError("Team.IsValid", "model.team.is_valid.reserved.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if !IsValidTeamName(o.Name) {
		return NewAppError("Team.IsValid", "model.team.is_valid.characters.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if len(o.AllowedDomains) > TEAM_ALLOWED_DOMAINS_MAX_LENGTH {
		return NewAppError("Team.IsValid", "model.team.is_valid.domains.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	return nil
}

func (o *Team) PreSave() {
	if o.Id == "" {
		o.Id = NewId()
	}

	if o.Type == "" {
		o.Type = TEAM_TYPE_PRIVATE
	}

	o.CreateAt = GetMillis()
	o.UpdateAt = o.CreateAt

	o.Name = SanitizeUnicode(o.Name)
	o.Description = SanitizeUnicode(o.Description)

	if len(o.InviteId) == 0 {
		o.InviteId = NewId()
	}
}

func (o *Team) PreUpdate() {
	o.UpdateAt = GetMillis()
	o.Name = SanitizeUnicode(o.Name)
	o.Description = SanitizeUnicode(o.Description)
}

func (o *Team) SanitizeInput() {
	o.LastPictureUpdate = 0
	o.InviteId = ""
}

func TeamFromJson(data io.Reader) *Team {
	var o *Team
	json.NewDecoder(data).Decode(&o)
	return o
}

func (o *Team) ToJson() string {
	b, _ := json.Marshal(o)
	return string(b)
}

func TeamListToJson(t []*Team) string {
	b, _ := json.Marshal(t)
	return string(b)
}

var reservedName = []string{
	"admin",
	"api",
	"error",
}

func (o *Team) Sanitize() {
	o.Email = ""
	o.InviteId = ""
}

func IsReservedTeamName(s string) bool {
	s = strings.ToLower(s)

	for _, value := range reservedName {
		if strings.Index(s, value) == 0 {
			return true
		}
	}

	return false
}

func IsValidTeamName(s string) bool {
	if !IsValidAlphaNum(s) {
		return false
	}

	if len(s) < TEAM_NAME_MIN_LENGTH {
		return false
	}

	return true
}

func CreateTeamImageKey(teamId string, time int64) string {
	etag := strconv.FormatInt(time, 10)
	path := "/teams/" + teamId + "/" + etag + ".jpg"
	return path
}

func (o *Team) GetTeamImageLink(settings *FileSettings) string {
	if o.LastPictureUpdate == 0 || *settings.AmazonCloudFrontURL == "" {
		return ""
	}

	path := (*settings.AmazonCloudFrontURL + CreateTeamImageKey(o.Id, o.LastPictureUpdate))
	return path
}
