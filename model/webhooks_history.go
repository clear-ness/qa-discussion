package model

import (
	"encoding/json"
)

type WebhooksHistory struct {
	Id             string `db:"Id, primarykey" json:"id"`
	WebhookId      string `db:"WebhookId" json:"webhook_id"`
	PostId         string `db:"PostId" json:"post_id"`
	TeamId         string `db:"TeamId" json:"team_id"`
	WebhookName    string `db:"WebhookName" json:"webhook_name"`
	URL            string `db:"URL" json:"name"`
	ContentType    string `db:"ContentType" json:"content_type"`
	RequestBody    string `db:"RequestBody" json:"request_body"`
	ResponseBody   string `db:"ResponseBody" json:"response_body"`
	ResponseStatus int    `db:"ResponseStatus" json:"response_status"`
	CreateAt       int64  `db:"CreateAt" json:"create_at"`
}

func WebhooksHistoryListToJson(list []*WebhooksHistory) string {
	b, _ := json.Marshal(list)
	return string(b)
}
