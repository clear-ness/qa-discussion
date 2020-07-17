package model

import (
	"encoding/json"
	"io"
	"net/http"
)

const (
	INBOX_MESSAGE_TYPE_ANSWER        = "answer"
	INBOX_MESSAGE_TYPE_COMMENT       = "comment"
	INBOX_MESSAGE_TYPE_COMMENT_REPLY = "comment_reply"

	INBOX_MESSAGE_CONTENT_MAX_LENGTH = 50
)

type InboxMessage struct {
	Id         string `db:"Id, primarykey" json:"id"`
	Type       string `db:"Type" json:"type"`
	Content    string `db:"Content" json:"content"`
	UserId     string `db:"UserId" json:"user_id"`
	SenderId   string `db:"SenderId" json:"sender_id,omitempty"`
	QuestionId string `db:"QuestionId" json:"question_id"`
	Title      string `db:"Title" json:"title"`
	AnswerId   string `db:"AnswerId" json:"answer_id,omitempty"`
	CommentId  string `db:"CommentId" json:"comment_id,omitempty"`
	CreateAt   int64  `db:"CreateAt" json:"create_at"`

	IsUnread bool `db:"-" json:"is_unread"`
}

func (o *InboxMessage) PreSave() {
	if o.Id == "" {
		o.Id = NewId()
	}

	if o.CreateAt == 0 {
		o.CreateAt = GetMillis()
	}
}

func (o *InboxMessage) IsValid() *AppError {
	if len(o.Content) > INBOX_MESSAGE_CONTENT_MAX_LENGTH || len(o.Content) <= 0 {
		return NewAppError("InboxMessage.IsValid", "model.inbox_message.is_valid.content.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.UserId) != 26 {
		return NewAppError("InboxMessage.IsValid", "model.inbox_message.is_valid.user_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.SenderId) != 26 {
		return NewAppError("InboxMessage.IsValid", "model.inbox_message.is_valid.sender_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.QuestionId) != 26 {
		return NewAppError("InboxMessage.IsValid", "model.inbox_message.is_valid.question_id.app_error", nil, "", http.StatusBadRequest)
	}

	if o.AnswerId != "" && len(o.AnswerId) != 26 {
		return NewAppError("InboxMessage.IsValid", "model.inbox_message.is_valid.answer_id.app_error", nil, "", http.StatusBadRequest)
	}

	if o.CommentId != "" && len(o.CommentId) != 26 {
		return NewAppError("InboxMessage.IsValid", "model.inbox_message.is_valid.comment_id.app_error", nil, "", http.StatusBadRequest)
	}

	if o.CreateAt == 0 {
		return NewAppError("InboxMessage.IsValid", "model.inbox_message.is_valid.create_at.app_error", nil, "", http.StatusBadRequest)
	}

	return nil
}

type InboxMessages []*InboxMessage

func InboxMessageListToJson(m []*InboxMessage) string {
	b, _ := json.Marshal(m)
	return string(b)
}

func InboxMessagesFromJson(data io.Reader) InboxMessages {
	var o InboxMessages
	json.NewDecoder(data).Decode(&o)
	return o
}
