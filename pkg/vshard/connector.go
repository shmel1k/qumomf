package vshard

import (
	"context"

	"github.com/viciious/go-tarantool"
)

type Connector struct {
	cfg  InstanceConfig
	conn *tarantool.Connector
}

func (c *Connector) Exec(ctx context.Context, q tarantool.Query, opts ...tarantool.ExecOption) *tarantool.Result {
	conn, err := c.conn.Connect()
	if err != nil {
		return &tarantool.Result{
			Error: err,
		}
	}

	return conn.Exec(ctx, q, opts...)
}

func (c *Connector) Close() {
	c.conn.Close()
}
