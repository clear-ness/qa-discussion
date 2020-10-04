package app

import (
	"github.com/clear-ness/qa-discussion/model"
)

type LineImportData struct {
	// Typeでどのモデルかを判断する
	Type string          `json:"type"`
	Team *TeamImportData `json:"team,omitempty"`
	User *UserImportData `json:"user,omitempty"`
	Post *PostImportData `json:"post,omitempty"`
}

type TeamImportData struct {
	Name            *string `json:"name"`
	DisplayName     *string `json:"display_name"`
	Type            *string `json:"type"`
	Description     *string `json:"description,omitempty"`
	AllowOpenInvite *bool   `json:"allow_open_invite,omitempty"`
	Scheme          *string `json:"scheme,omitempty"`
}

type UserImportData struct {
	ProfileImage       *string `json:"profile_image,omitempty"`
	Username           *string `json:"username"`
	Email              *string `json:"email"`
	AuthService        *string `json:"auth_service"`
	AuthData           *string `json:"auth_data,omitempty"`
	Password           *string `json:"password,omitempty"`
	Nickname           *string `json:"nickname"`
	FirstName          *string `json:"first_name"`
	LastName           *string `json:"last_name"`
	Position           *string `json:"position"`
	Roles              *string `json:"roles"`
	Locale             *string `json:"locale"`
	UseMarkdownPreview *string `json:"feature_enabled_markdown_preview,omitempty"`
	UseFormatting      *string `json:"formatting,omitempty"`
	ShowUnreadSection  *string `json:"show_unread_section,omitempty"`
	DeleteAt           *int64  `json:"delete_at,omitempty"`

	Teams *[]UserTeamImportData `json:"teams,omitempty"`

	Theme              *string `json:"theme,omitempty"`
	UseMilitaryTime    *string `json:"military_time,omitempty"`
	CollapsePreviews   *string `json:"link_previews,omitempty"`
	MessageDisplay     *string `json:"message_display,omitempty"`
	ChannelDisplayMode *string `json:"channel_display_mode,omitempty"`
	TutorialStep       *string `json:"tutorial_step,omitempty"`
	EmailInterval      *string `json:"email_interval,omitempty"`
}

type UserTeamImportData struct {
	Name *string `json:"name"`
	Type *string `json:"type"`
}

type PostImportData struct {
	Team *string `json:"team"`
	User *string `json:"user"`

	Content  *string                `json:"content"`
	Props    *model.StringInterface `json:"props"`
	CreateAt *int64                 `json:"create_at"`

	FlaggedBy   *[]string               `json:"flagged_by,omitempty"`
	Reactions   *[]ReactionImportData   `json:"reactions,omitempty"`
	Replies     *[]ReplyImportData      `json:"replies,omitempty"`
	Attachments *[]AttachmentImportData `json:"attachments,omitempty"`
}

// 受信データを複数goルーチンで並列処理するため
type LineImportWorkerData struct {
	LineImportData
	LineNumber int
}

// 受信データを複数goルーチンで並列処理するため
type LineImportWorkerError struct {
	Error      *model.AppError
	LineNumber int
}
