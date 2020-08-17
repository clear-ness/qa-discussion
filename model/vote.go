package model

import (
	"encoding/json"
	"io"
)

const (
	// TODO: accepts、delete by moderator
	VOTE_TYPE_UP_VOTE   = "up_vote"
	VOTE_TYPE_DOWN_VOTE = "down_vote"
	VOTE_TYPE_FLAG      = "flag"
)

type Vote struct {
	PostId   string `db:"PostId" json:"post_id"`
	UserId   string `db:"UserId" json:"user_id"`
	Type     string `db:"Type" json:"type"`
	TeamId   string `db:"TeamId" json:"team_id"`
	CreateAt int64  `db:"CreateAt" json:"create_at"`
}

type VoteWithPost struct {
	*Vote
	Post *Post `json:"post"`
}

type VotesWithCount struct {
	Votes      []*VoteWithPost `json:"votes"`
	TotalCount int64           `json:"total_count"`
}

func (o *VotesWithCount) ToJson() []byte {
	b, _ := json.Marshal(o)
	return b
}

func VotesToJson(v []*Vote) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func VotesWithCountFromJson(data io.Reader) *VotesWithCount {
	var o *VotesWithCount
	json.NewDecoder(data).Decode(&o)
	return o
}
