package vshard

import (
	"time"

	"github.com/shmel1k/qumomf/internal/config"
	"github.com/shmel1k/qumomf/internal/storage"
	"github.com/shmel1k/qumomf/internal/util"
)

func MockCluster() *Cluster {
	return NewCluster("sandbox", storage.MockedStorage{}, config.ClusterConfig{
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
		Priorities: map[string]int{
			"bd64dd00-161e-4c99-8b3c-d3c4635e18d2": 10,
			"cc4cfb9c-11d8-4810-84d2-66cfbebb0f6e": 5,
		},
		Routers: []config.RouterConfig{
			{
				Name: "router_1",
				Addr: "127.0.0.1:9301",
			},
		},
	})
}
