package model

import (
	"encoding/json"
)

type CollectionList []*Collection

func (o *CollectionList) ToJson() string {
	if b, err := json.Marshal(o); err != nil {
		return "[]"
	} else {
		return string(b)
	}
}
