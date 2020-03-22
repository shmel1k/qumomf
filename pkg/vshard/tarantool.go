package vshard

import (
	"github.com/viciious/go-tarantool"

	"github.com/shmel1k/qumomf/internal/config"
)

func setupConnection(c config.InstanceConfig) *Connector {
	cfg := &tarantool.Options{
		ConnectTimeout: *c.ConnectTimeout,
		QueryTimeout:   *c.RequestTimeout,
		Password:       *c.Password,
		User:           *c.User,
		UUID:           c.UUID,
	}

	conn := tarantool.New(c.Addr, cfg)
	return &Connector{
		conn: conn,
	}
}
