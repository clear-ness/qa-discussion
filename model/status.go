package model

import (
	"encoding/json"
	"io"
)

const (
	STATUS_OFFLINE = "offline"
	STATUS_AWAY    = "away"
	STATUS_ONLINE  = "online"

	// LastActivityAtを更新するのに必要な最低経過時間
	STATUS_MIN_UPDATE_TIME = 120000 // 2 minutes
)

type Status struct {
	UserId         string `json:"user_id"`
	Status         string `json:"status"`
	Manual         bool   `json:"manual"`
	LastActivityAt int64  `json:"last_activity_at"`
	ActiveTeam     string `json:"active_team,omitempty" db:"-"`
}

func (o *Status) ToJson() string {
	oCopy := *o
	oCopy.ActiveTeam = ""
	b, _ := json.Marshal(oCopy)
	return string(b)
}

func StatusListToJson(u []*Status) string {
	uCopy := make([]Status, len(u))

	for i, s := range u {
		sCopy := *s
		sCopy.ActiveTeam = ""
		uCopy[i] = sCopy
	}

	b, _ := json.Marshal(uCopy)
	return string(b)
}

func StatusFromJson(data io.Reader) *Status {
	var o *Status
	json.NewDecoder(data).Decode(&o)
	return o
}
