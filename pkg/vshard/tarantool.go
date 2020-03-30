package vshard

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/viciious/go-tarantool"
)

type ConnPool interface {
	Get(uri, uuid string) *Connector
	Close()
}

type ConnOptions struct {
	User           string
	Password       string
	UUID           string
	ConnectTimeout time.Duration
	QueryTimeout   time.Duration
}

type OverrideURIRules map[string]string

type pool struct {
	template ConnOptions
	rules    OverrideURIRules

	m     map[string]*Connector
	mutex sync.RWMutex
}

func NewConnPool(template ConnOptions, rules OverrideURIRules) ConnPool {
	return &pool{
		template: template,
		rules:    rules,
		m:        make(map[string]*Connector),
	}
}

func (p *pool) Get(uri, uuid string) *Connector {
	u := removeUserInfo(uri)
	u = overrideURI(u, p.rules)

	p.mutex.RLock()
	conn, ok := p.m[u]
	p.mutex.RUnlock()
	if ok {
		return conn
	}

	p.mutex.Lock()
	opts := p.template
	opts.UUID = uuid
	conn = setupConnection(u, opts)
	p.m[u] = conn
	p.mutex.Unlock()

	return conn
}

func overrideURI(uri string, rules OverrideURIRules) string {
	u, ok := rules[uri]
	if ok {
		return u
	}
	return uri
}

func (p *pool) Close() {
	p.mutex.Lock()
	for _, conn := range p.m {
		conn.Close()
	}
	p.mutex.Unlock()
}

func removeUserInfo(uri string) string {
	if idx := strings.IndexByte(uri, '@'); idx >= 0 {
		return uri[idx+1:]
	}
	return uri
}

type Connector struct {
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

func setupConnection(uri string, c ConnOptions) *Connector {
	cfg := &tarantool.Options{
		User:           c.User,
		Password:       c.Password,
		UUID:           c.UUID,
		ConnectTimeout: c.ConnectTimeout,
		QueryTimeout:   c.QueryTimeout,
	}

	conn := tarantool.New(uri, cfg)
	return &Connector{
		conn: conn,
	}
}
