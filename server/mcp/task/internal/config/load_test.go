package config

import (
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

const sampleConfig = `RpcServerConf:
  Name: task.rpc
  ListenOn: 0.0.0.0:50050
  Mode: dev

TaskRedis:
  Host: 127.0.0.1:6379
  Type: node
  Pass: ""

LocalStoragePath: ./data/videos
`

func TestLoadConfig(t *testing.T) {
	var cfg Config
	if err := conf.LoadFromYamlBytes([]byte(sampleConfig), &cfg); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
}











