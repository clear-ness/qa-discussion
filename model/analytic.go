package model

import (
	"encoding/json"
	"io"
)

type Analytic struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type Analytics []*Analytic

func (me *Analytic) ToJson() string {
	b, _ := json.Marshal(me)
	return string(b)
}

func AnalyticFromJson(data io.Reader) *Analytic {
	var me *Analytic
	json.NewDecoder(data).Decode(&me)
	return me
}

func (me Analytics) ToJson() string {
	if b, err := json.Marshal(me); err != nil {
		return "[]"
	} else {
		return string(b)
	}
}

func AnalyticsFromJson(data io.Reader) Analytics {
	var me Analytics
	json.NewDecoder(data).Decode(&me)
	return me
}
