package main

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/tarantool/go-tarantool"
)

const (
	getCall = "qumomf_get"
	setCall = "qumomf_set"
)

const (
	routerAddr = "127.0.0.1:9301"
	user       = "qumomf"
	pass       = "qumomf"
	stringLen  = 5
)

func ParseString(f interface{}) (string, error) {
	switch t := f.(type) {
	case string:
		return t, nil
	}
	return "", fmt.Errorf("got invalid type %T for value %v", f, f)
}

func parseResponse(data []interface{}) (string, error) {
	if len(data) == 0 {
		return "", nil
	}

	d := data[0].([]interface{})
	value, err := ParseString(d[0])
	if err != nil {
		return "", err
	}

	return value, nil
}

func GetRandomString(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func Test_Router_AddAndCheckKey(t *testing.T) {
	time.Sleep(time.Second)
	opts := tarantool.Opts{
		User:    user,
		Pass:    pass,
		Timeout: 3 * time.Second,
	}

	conn, err := tarantool.Connect(routerAddr, opts)
	assert.Nil(t, err)

	_, err = conn.Ping()
	assert.Nil(t, err)

	expectedKey := GetRandomString(stringLen)
	expectedValue := GetRandomString(stringLen)

	resp, err := conn.Call(setCall, []interface{}{expectedKey, expectedValue})
	assert.Nil(t, err)

	resp, err = conn.Call(getCall, []interface{}{expectedKey})
	assert.Nil(t, err)

	value, err := parseResponse(resp.Data)
	assert.Nil(t, err)
	assert.Equal(t, expectedValue, value)
}
