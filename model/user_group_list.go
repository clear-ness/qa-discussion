package model

import (
	"encoding/json"
)

type UserGroupList []*UserGroup

func (o *UserGroupList) ToJson() string {
	if b, err := json.Marshal(o); err != nil {
		return "[]"
	} else {
		return string(b)
	}
}
