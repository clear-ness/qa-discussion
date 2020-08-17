package model

import (
	"encoding/json"
	"io"
)

type Audits []Audit

func (o Audits) ToJson() string {
	if b, err := json.Marshal(o); err != nil {
		return "[]"
	} else {
		return string(b)
	}
}

func AuditsFromJson(data io.Reader) Audits {
	var o Audits
	json.NewDecoder(data).Decode(&o)
	return o
}
