package model

type GroupMemberHistory struct {
	GroupId   string `db:"GroupId"`
	UserId    string `db:"UserId"`
	JoinTime  int64  `db:"JoinTime"`
	LeaveTime *int64 `db:"LeaveTime"`
}
