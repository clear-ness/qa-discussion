package model

import (
	"encoding/json"
)

const (
	// TODO: more types
	USER_POINT_TYPE_CREATE_QUESTION     = "create_question"
	USER_POINT_TYPE_CREATE_ANSWER       = "create_answer"
	USER_POINT_TYPE_SELECT_ANSWER       = "select_answer"
	USER_POINT_TYPE_SELECTED_ANSWER     = "selected_answer"
	USER_POINT_TYPE_DELETE_QUESTION     = "delete_question"
	USER_POINT_TYPE_DELETE_ANSWER       = "delete_answer"
	USER_POINT_TYPE_VOTED               = "voted"
	USER_POINT_TYPE_VOTED_CANCELED      = "voted_canceled"
	USER_POINT_TYPE_DOWN_VOTED          = "down_voted"
	USER_POINT_TYPE_DOWN_VOTED_CANCELED = "down_voted_canceled"
	USER_POINT_TYPE_FLAGGED             = "flagged"
	USER_POINT_TYPE_FLAGGED_CANCELED    = "flagged_canceled"

	USER_POINT_FOR_CREATE_QUESTION = 3
	USER_POINT_FOR_CREATE_ANSWER   = 3
	USER_POINT_FOR_SELECT_ANSWER   = 3
	USER_POINT_FOR_SELECTED_ANSWER = 10
	USER_POINT_FOR_VOTED           = 5
	USER_POINT_FOR_DOWN_VOTED      = -3
	USER_POINT_FOR_FLAGGED         = -2

	MIN_USER_POINT_FOR_ANSWER_FOR_PROTECTED_POST = 10
	MIN_USER_POINT_FOR_VOTE_REVIEW               = 100

	USER_POINT_HISTORY_INTERVAL_DAY   = "day"
	USER_POINT_HISTORY_INTERVAL_WEEK  = "week"
	USER_POINT_HISTORY_INTERVAL_MONTH = "month"
)

type UserPointHistory struct {
	Id       string `db:"Id, primarykey" json:"id"`
	TeamId   string `db:"TeamId" json:"team_id"`
	UserId   string `db:"UserId" json:"user_id"`
	Type     string `db:"Type" json:"type"`
	PostId   string `db:"PostId" json:"post_id"`
	PostType string `db:"PostType" json:"post_type"`
	Tags     string `db:"Tags" json:"tags,omitempty"`
	Points   int    `db:"Points" json:"points"`
	CreateAt int64  `db:"CreateAt" json:"create_at"`
}

func UserPointHistoryToJson(u []*UserPointHistory) string {
	b, _ := json.Marshal(u)
	return string(b)
}

type TopUserByTagResult struct {
	UserId     string `json:"user_id"`
	TotalScore int    `json:"total_score"`
}

type TopPostByTagResult struct {
	PostId     string `json:"post_id"`
	TotalScore int    `json:"total_score"`
}

type TopUserByTagResultsWithCount struct {
	TopUserByTagResults []*TopUserByTagResult `json:"top_user_by_tag_result"`
	TotalCount          int64                 `json:"total_count"`
}

func (o *TopUserByTagResultsWithCount) ToJson() []byte {
	b, _ := json.Marshal(o)
	return b
}

type TopPostByTagResultsWithCount struct {
	TopPostByTagResults []*TopPostByTagResult `json:"top_post_by_tag_result"`
	TotalCount          int64                 `json:"total_count"`
}

func (o *TopPostByTagResultsWithCount) ToJson() []byte {
	b, _ := json.Marshal(o)
	return b
}
