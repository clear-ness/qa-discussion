package model

const (
	POST_COUNTER_MAX = 5
)

type PostViewsHistory struct {
	Id         string `db:"Id, primarykey" json:"id"`
	PostId     string `db:"PostId" json:"post_id"`
	TeamId     string `db:"TeamId" json:"team_id"`
	UserId     string `db:"UserId" json:"user_id"`
	IpAddress  string `db:"IpAddress" json:"ip_address"`
	ViewsCount int    `db:"ViewsCount" json:"views_count"`
	CreateAt   int64  `db:"CreateAt" json:"create_at"`
}
