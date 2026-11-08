package svc

import (
	"net"
	"testing"

	"go-zero-admin/application/ai/rpc/internal/config"
)

func TestConfiguredVectorStoreSurvivesStartupOutage(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	var c config.Config
	c.Qdrant.Address = address
	c.AI.EmbeddingProvider = "qwen"
	c.AI.EmbeddingModel = "fixture"
	c.AI.EmbeddingDimensions = 4
	store := newVectorStore(c)
	if store == nil {
		t.Fatal("已配置的向量存储遇到启动暂态故障应保留恢复入口")
	}
	t.Cleanup(func() { _ = store.Close() })
}
