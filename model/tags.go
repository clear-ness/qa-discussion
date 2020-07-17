package model

import (
	"encoding/json"
	"io"
)

type Tags []Tag

func (o *Tags) ToJson() string {
	b, _ := json.Marshal(o)
	return string(b)
}

func TagsFromJson(data io.Reader) (Tags, error) {
	decoder := json.NewDecoder(data)
	var o Tags
	err := decoder.Decode(&o)
	if err == nil {
		return o, nil
	} else {
		return nil, err
	}
}
