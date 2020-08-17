package model

type TeamMemberHistory struct {
	TeamId    string `db:"TeamId"`
	UserId    string `db:"UserId"`
	JoinTime  int64  `db:"JoinTime"`
	LeaveTime *int64 `db:"LeaveTime"`
}
