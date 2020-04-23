package vshard

import (
	"time"

	"github.com/shmel1k/qumomf/internal/config"
	"github.com/shmel1k/qumomf/pkg/util"
)

func MockCluster() *Cluster {
	return NewCluster("sandbox", config.ClusterConfig{
		Connection: &config.ConnectConfig{
			User:           util.NewString("qumomf"),
			Password:       util.NewString("qumomf"),
			ConnectTimeout: util.NewDuration(1 * time.Second),
			RequestTimeout: util.NewDuration(1 * time.Second),
		},
		ReadOnly: util.NewBool(true),
		OverrideURIRules: map[string]string{
			"qumomf_1_m.ddk:3301":   "127.0.0.1:9303",
			"qumomf_1_s.ddk:3301":   "127.0.0.1:9304",
			"qumomf_2_m.ddk:3301":   "127.0.0.1:9305",
			"qumomf_2_s_1.ddk:3301": "127.0.0.1:9306",
			"qumomf_2_s_2.ddk:3301": "127.0.0.1:9307",
		},
		Routers: []config.RouterConfig{
			{
				Name: "router_1",
				Addr: "127.0.0.1:9301",
				UUID: "router_uuid_1",
			},
		},
	})
}
