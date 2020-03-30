package vshard

import "time"

func timestamp() int64 {
	return time.Now().UTC().Unix()
}
