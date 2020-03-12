package main

import (
	"math/rand"
	"testing"
	"time"

	"github.com/shmel1k/qumomf/example/LuaHelpers"

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

type tntAnswer struct {
	Key   string
	Value string
}

func parseResponse(data []interface{}) (string, error) {
	if len(data) == 0 {
		return "", nil
	}

	d := data[0].([]interface{})
	value, err := LuaHelpers.ParseString(d[0])
	if err != nil {
		return "", err
	}

	return value, nil
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func Test_Router_AddAndCheckKey(t *testing.T) {
	opts := tarantool.Opts{
		User:    user,
		Pass:    pass,
		Timeout: 3 * time.Second,
	}

	conn, err := tarantool.Connect(routerAddr, opts)
	assert.Nil(t, err)

	_, err = conn.Ping()
	assert.Nil(t, err)

	keyTest1 := RandStringRunes(stringLen)
	valueTest1 := RandStringRunes(stringLen)

	_, err = conn.Call(setCall, []interface{}{keyTest1, valueTest1})
	assert.Nil(t, err)

	resp, err := conn.Call(getCall, []interface{}{keyTest1})
	assert.Nil(t, err)

	value, err := parseResponse(resp.Data)
	assert.Nil(t, err)
	assert.Equal(t, valueTest1, value)
}
