package vshard

import "time"

func timestamp() int64 {
	return time.Now().UTC().Unix()
}

func newBool(v bool) *bool {
	return &v
}

func newDuration(v time.Duration) *time.Duration {
	return &v
}

func newString(v string) *string {
	return &v
}
