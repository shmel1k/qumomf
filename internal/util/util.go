package util

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

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

func GetHash(in []byte) (string, error) {
	h := sha256.New()
	_, err := h.Write(in)

	return hex.EncodeToString(h.Sum(nil)), err
}
