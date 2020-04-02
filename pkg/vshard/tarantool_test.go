package vshard

import (
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_removeUserInfo(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string
	}{
		{
			name: "NoUserInfo_ShouldReturnTheSameUri",
			uri:  "tarantool.repl:3301",
			want: "tarantool.repl:3301",
		},
		{
			name: "Username_ShouldReturnHostAndPort",
			uri:  "qumomf@tarantool.repl:3301",
			want: "tarantool.repl:3301",
		},
		{
			name: "UsernameAndPass_ShouldReturnHostAndPort",
			uri:  "qumomf:qumomf@tarantool.repl:3301",
			want: "tarantool.repl:3301",
		},
	}
	for _, tv := range tests {
		tt := tv
		t.Run(tt.name, func(t *testing.T) {
			got := removeUserInfo(tt.uri)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_overrideURI(t *testing.T) {
	type args struct {
		uri   string
		rules OverrideURIRules
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "NoRules_ShouldReturnTheSameUri",
			args: args{
				uri:   "tarantool.repl:3301",
				rules: nil,
			},
			want: "tarantool.repl:3301",
		},
		{
			name: "NoSuitableRule_ShouldReturnTheSameUri",
			args: args{
				uri: "tarantool.repl:3301",
				rules: OverrideURIRules{
					"tarantool2.repl:3301": "tnt2.repl:3301",
					"tarantool.repl:8801":  "tnt.repl:8801",
				},
			},
			want: "tarantool.repl:3301",
		},
		{
			name: "RuleApplied_ShouldReturnOverridden",
			args: args{
				uri: "tarantool.repl:3301",
				rules: OverrideURIRules{
					"tarantool.repl:3301": "tnt.repl:3301",
					"tarantool.repl:8801": "tnt.repl:8801",
				},
			},
			want: "tnt.repl:3301",
		},
	}
	for _, tv := range tests {
		tt := tv
		t.Run(tt.name, func(t *testing.T) {
			got := overrideURI(tt.args.uri, tt.args.rules)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPool_Get(t *testing.T) {
	connOpts := ConnOptions{
		User:     "qumomf",
		Password: "qumomf",
		UUID:     "uuid",
	}
	p := NewConnPool(connOpts, nil)
	uri := "tarantool.repl:3301"
	n := 1000

	ch := make(chan *Connector, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			ch <- p.Get(uri, "uuid")
			wg.Done()
		}()
	}
	wg.Wait()
	close(ch)

	var conn *Connector
	for c := range ch {
		if conn == nil {
			conn = c
		}
		require.Same(t, conn, c)
	}

	p.Close()
}

func BenchmarkPool_Get(b *testing.B) {
	connOpts := ConnOptions{
		User:     "qumomf",
		Password: "qumomf",
		UUID:     "uuid",
	}
	p := NewConnPool(connOpts, nil)

	var ub strings.Builder
	var uri string
	var conn *Connector

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ub.Reset()
		ub.WriteString("tnt-")
		ub.WriteString(strconv.Itoa(i))
		ub.WriteString(":3301")
		uri = ub.String()

		conn = p.Get(uri, "uuid")
		conn.Close()
	}
}
