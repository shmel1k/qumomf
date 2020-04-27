package util

import "time"

func Timestamp() int64 {
	return time.Now().Unix()
}

func NewBool(v bool) *bool {
	return &v
}

func NewDuration(v time.Duration) *time.Duration {
	return &v
}

func NewString(v string) *string {
	return &v
}
