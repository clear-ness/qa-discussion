package model

import (
	"encoding/json"
)

type GroupList []*Group

func (o *GroupList) ToJson() string {
	if b, err := json.Marshal(o); err != nil {
		return "[]"
	} else {
		return string(b)
	}
}
