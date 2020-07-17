package model

import (
	"encoding/json"
	"io"
	"net/http"
)

const (
	NOTIFICATION_INBOX_INTERVAL_THREE_HOUR = "three_hour"
	NOTIFICATION_INBOX_INTERVAL_DAY        = "day"
	NOTIFICATION_INBOX_INTERVAL_WEEK       = "week"
)

type NotificationSetting struct {
	Id            string `db:"Id, primarykey" json:"id"`
	UserId        string `db:"UserId" json:"user_id"`
	InboxInterval string `db:"InboxInterval" json:"inbox_interval"`
	CreateAt      int64  `db:"CreateAt" json:"create_at"`
	UpdateAt      int64  `db:"UpdateAt" json:"update_at"`
}

func (o *NotificationSetting) ToJson() string {
	b, _ := json.Marshal(o)
	return string(b)
}

func NotificationSettingFromJson(data io.Reader) *NotificationSetting {
	var o *NotificationSetting
	json.NewDecoder(data).Decode(&o)
	return o
}

func (o *NotificationSetting) PreSave() {
	if o.Id == "" {
		o.Id = NewId()
	}

	if o.CreateAt == 0 {
		o.CreateAt = GetMillis()
	}

	o.UpdateAt = o.CreateAt
}

func (o *NotificationSetting) PreUpdate() {
	o.UpdateAt = GetMillis()
}

func (o *NotificationSetting) IsValid() *AppError {
	if len(o.Id) != 26 {
		return NewAppError("NotificationSetting.IsValid", "model.notification_setting.is_valid.id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.UserId) != 26 {
		return NewAppError("NotificationSetting.IsValid", "model.notification_setting.is_valid.user_id.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if o.CreateAt == 0 {
		return NewAppError("NotificationSetting.IsValid", "model.notification_setting.is_valid.create_at.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if len(o.InboxInterval) != 0 && o.InboxInterval != NOTIFICATION_INBOX_INTERVAL_THREE_HOUR && o.InboxInterval != NOTIFICATION_INBOX_INTERVAL_DAY && o.InboxInterval != NOTIFICATION_INBOX_INTERVAL_WEEK {
		return NewAppError("NotificationSetting.IsValid", "model.notification_setting.is_valid.Interval.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	return nil
}
