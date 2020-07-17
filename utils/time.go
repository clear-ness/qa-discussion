package utils

import (
	"time"
)

func MillisFromTime(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}
