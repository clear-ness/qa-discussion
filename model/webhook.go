package model

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type Webhook struct {
	Id             string      `db:"Id, primarykey" json:"id"`
	Token          string      `db:"Token" json:"token"`
	UserId         string      `db:"UserId" json:"user_id"`
	TeamId         string      `db:"TeamId" json:"team_id"`
	QuestionEvents bool        `db:"QuestionEvents" json:"question_events"`
	AnswerEvents   bool        `db:"AnswerEvents" json:"answer_events"`
	CommentEvents  bool        `db:"CommentEvents" json:"comment_events"`
	URLs           StringArray `db:"URLs" json:"urls"`
	Name           string      `db:"Name" json:"name"`
	Description    string      `db:"Description" json:"description"`
	ContentType    string      `db:"ContentType" json:"content_type"`
	CreateAt       int64       `db:"CreateAt" json:"create_at"`
	UpdateAt       int64       `db:"UpdateAt" json:"update_at"`
	DeleteAt       int64       `db:"DeleteAt" json:"delete_at"`
}

func (o *Webhook) ToJson() string {
	b, _ := json.Marshal(o)
	return string(b)
}

func WebhookFromJson(data io.Reader) *Webhook {
	var o *Webhook
	json.NewDecoder(data).Decode(&o)
	return o
}

func WebhookListToJson(list []*Webhook) string {
	b, _ := json.Marshal(list)
	return string(b)
}

func (o *Webhook) IsValid() *AppError {
	if len(o.Id) != 26 {
		return NewAppError("Webhook.IsValid", "model.hook.is_valid.id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.Token) != 26 {
		return NewAppError("Webhook.IsValid", "model.hook.is_valid.token.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.UserId) != 26 {
		return NewAppError("Webhook.IsValid", "model.hook.is_valid.user_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.TeamId) != 26 {
		return NewAppError("Webhook.IsValid", "model.hook.is_valid.team_id.app_error", nil, "", http.StatusBadRequest)
	}

	if !o.QuestionEvents && !o.AnswerEvents && !o.CommentEvents {
		return NewAppError("Webhook.IsValid", "model.hook.is_valid.events.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.URLs) == 0 || len(fmt.Sprintf("%s", o.URLs)) > 1024 {
		return NewAppError("Webhook.IsValid", "model.hook.is_valid.urls.app_error", nil, "", http.StatusBadRequest)
	}
	for _, url := range o.URLs {
		if !IsValidHttpUrl(url) {
			return NewAppError("Webhook.IsValid", "model.hook.is_valid.url.app_error", nil, "", http.StatusBadRequest)
		}
	}

	if len(o.Name) > 64 {
		return NewAppError("Webhook.IsValid", "model.hook.is_valid.name.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.Description) > 255 {
		return NewAppError("Webhook.IsValid", "model.hook.is_valid.description.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.ContentType) > 128 {
		return NewAppError("Webhook.IsValid", "model.hook.is_valid.content_type.app_error", nil, "", http.StatusBadRequest)
	}

	if o.CreateAt == 0 {
		return NewAppError("Webhook.IsValid", "model.hook.is_valid.create_at.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if o.UpdateAt == 0 {
		return NewAppError("Webhook.IsValid", "model.hook.is_valid.update_at.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	return nil
}

type WebhookPayload struct {
	Token    string `json:"token"`
	TeamId   string `json:"team_id"`
	TeamName string `json:"team_name"`
	// post.CreateAt
	Timestamp int64 `json:"timestamp"`
	// post.UserId(投稿の作者)
	UserId string `json:"user_id"`
	// post.User.Name(投稿の作者の名前)
	UserName string `json:"user_name"`
	PostId   string `json:"post_id"`
	// post.Content
	Content string `json:"content"`
	// 質問の場合はそのタイトル、
	// 回答またはコメントの場合はroot(質問)のタイトル。
	Title    string `json:"title"`
	PostType string `json:"post_type"`
}

func (o *WebhookPayload) ToJSON() string {
	b, _ := json.Marshal(o)
	return string(b)
}

func (o *WebhookPayload) ToFormValues() string {
	// キーがcase sensitiveな map[string][]string
	v := url.Values{}
	v.Set("token", o.Token)
	v.Set("team_id", o.TeamId)
	v.Set("team_name", o.TeamName)
	v.Set("timestamp", strconv.FormatInt(o.Timestamp/1000, 10))
	v.Set("user_id", o.UserId)
	v.Set("user_name", o.UserName)
	v.Set("post_id", o.PostId)
	v.Set("content", o.Content)
	v.Set("title", o.Title)
	v.Set("post_type", o.PostType)

	return v.Encode()
}

func (o *Webhook) PreSave() {
	if o.Id == "" {
		o.Id = NewId()
	}

	o.Token = NewId()

	o.CreateAt = GetMillis()
	o.UpdateAt = o.CreateAt

	o.DeleteAt = int64(0)
}

type WebhookResponse struct {
	Text *string `json:"text"`
}

func (o *WebhookResponse) ToJson() string {
	b, _ := json.Marshal(o)
	return string(b)
}

func WebhookResponseFromJson(data io.Reader) (*WebhookResponse, error) {
	var o *WebhookResponse
	err := json.NewDecoder(data).Decode(&o)
	return o, err
}
