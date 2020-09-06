package model

import (
	"encoding/json"
	"io"
)

const (
	// TODO: accepts
	VOTE_TYPE_UP_VOTE   = "up_vote"
	VOTE_TYPE_DOWN_VOTE = "down_vote"

	// 一般ユーザー向け(flag)、reputationユーザー向け(review)、システム向け(system)
	// のそれぞれ3種類のreview voteがあり得る。
	// reopenについては一旦考えない、削除されたら削除されたまま。
	// 一旦teamに紐づくpostついては考えない、system postのみ。
	// (with no reasons)
	VOTE_TYPE_FLAG = "flag"
	// (with reasons) (e.g. abuse, spam, invalid, adult, duplicate)
	VOTE_TYPE_REVIEW = "review"
	// (with reasons, with no user) (e.g. new users posts, answer to old question)
	VOTE_TYPE_SYSTEM = "system"
)

type Vote struct {
	PostId       string `db:"PostId" json:"post_id"`
	UserId       string `db:"UserId" json:"user_id"`
	Type         string `db:"Type" json:"type"`
	Tags         string `db:"Tags" json:"tags,omitempty"`
	TeamId       string `db:"TeamId" json:"team_id"`
	FirstPostRev int    `db:"FirstPostRev" json:"first_post_rev"`
	LastPostRev  int    `db:"LastPostRev" json:"last_post_rev"`
	CreateAt     int64  `db:"CreateAt" json:"create_at"`
	InvalidateAt int64  `db:"InvalidateAt" json:"invalidate_at"`
	CompletedAt  int64  `db:"CompletedAt" json:"completed_at"`
	CompletedBy  string `db:"CompletedBy" json:"completed_by"`
	RejectedAt   int64  `db:"RejectedAt" json:"rejected_at"`
	RejectedBy   string `db:"RejectedBy" json:"rejected_by"`
}

func (o *Vote) Clone() *Vote {
	copy := *o
	return &copy
}

func (o *Vote) ToJson() string {
	copy := o.Clone()
	b, _ := json.Marshal(copy)
	return string(b)
}

type Reviews []Vote

type ReviewsWithCount struct {
	Reviews    []*Vote `json:"reviews"`
	TotalCount int64   `json:"total_count"`
}

func (o *ReviewsWithCount) ToJson() []byte {
	b, _ := json.Marshal(o)
	return b
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

// tag term only
type SearchReviewsOptions struct {
	Tagged             string
	UserId             string
	PostId             string
	ReviewType         string
	FromDate           int64
	ToDate             int64
	Page               int
	PerPage            int
	TeamId             string
	IncludeCompleted   bool
	IncludeRejected    bool
	IncludeInvalidated bool
}
