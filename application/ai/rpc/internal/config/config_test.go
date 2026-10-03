package config

import (
	"os"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/conf"
	"go-zero-admin/pkg/aiagent"
)

func TestLocalConfigDoesNotRequireBusinessRedisOrModelCredentials(t *testing.T) {
	var cfg Config
	if err := conf.Load("../../etc/ai.yaml", &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.ListenOn != "127.0.0.1:6002" || cfg.Etcd.Key != "ai.rpc" || cfg.AppletRPC.Etcd.Key != "applet.rpc" || cfg.BizRedis.Host != "" || cfg.DB.MaxOpenConns != 100 || cfg.DB.MaxIdleConns != 10 {
		t.Fatal("local AI config does not match the independent service contract")
	}
	if cfg.AI.Enabled || cfg.AI.Provider != aiagent.ProviderDeepSeek || cfg.AI.MaxInputChars != 2000 || cfg.AI.MaxSteps != 6 || cfg.AI.MaxRunSeconds != 90 {
		t.Fatal("local AI YAML parameters do not match the default contract")
	}
}

func TestYAMLSettingsReachRuntimeAndCredentialsStayInEnvironment(t *testing.T) {
	t.Setenv("QWEN_API_KEY", "fixture-qwen-key")
	t.Setenv("AI_AGENT_PROVIDER", "deepseek")
	t.Setenv("AI_AGENT_MAX_STEPS", "8")
	var cfg Config
	content := []byte(`Name: ai.rpc
ListenOn: 127.0.0.1:6002
AppletRPC:
  Target: 127.0.0.1:6001
DB:
  DataSource: fixture
AI:
  Enabled: true
  Provider: qwen
  MaxInputChars: 1000
  MaxSteps: 2
  MaxRunSeconds: 45
  Qwen:
    Model: fixture-model
    BaseURL: https://example.com/compatible-mode/v1
`)
	if err := conf.LoadFromYamlBytes(content, &cfg); err != nil {
		t.Fatal(err)
	}
	c, err := aiagent.LoadConfig(cfg.AI)
	if err != nil || !c.Enabled || c.Provider != "qwen" || c.Model != "fixture-model" || c.BaseURL != "https://example.com/compatible-mode/v1" || c.APIKey != "fixture-qwen-key" || c.MaxInputChars != 1000 || c.MaxSteps != 2 || c.RunTimeout != 45*time.Second {
		t.Fatal("runtime ignored YAML settings", c, err)
	}
}

func TestOptionalAIBlockRetainsDisabledStartup(t *testing.T) {
	content, err := os.ReadFile("../../etc/ai.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// Older configurations omitted AI entirely. Keep that deployment path valid.
	for i := 0; i+4 <= len(content); i++ {
		if string(content[i:i+4]) == "\nAI:" {
			content = content[:i]
			break
		}
	}
	var cfg Config
	if err := conf.LoadFromYamlBytes(content, &cfg); err != nil {
		t.Fatal(err)
	}
	c, err := aiagent.LoadConfig(cfg.AI)
	if err != nil || c.Enabled || c.MaxSteps != 6 || c.MaxInputChars != 2000 || c.MaxRunSeconds != 90 {
		t.Fatal("missing optional AI block impacted startup", c, err)
	}
}
