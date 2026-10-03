package aiagent

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestYAMLSettingsDefaultDisabledAndSelectProviderCredentials(t *testing.T) {
	values := map[string]string{"DEEPSEEK_API_KEY": "unused-key", "QWEN_API_KEY": "test-secret"}
	getenv := func(name string) string { return values[name] }
	c, err := loadConfig(Settings{}, func(string) string {
		t.Fatal("disabled feature read credentials")
		return ""
	})
	if err != nil || c.Enabled || c.APIKey != "" || c.MaxSteps != 6 || c.MaxInputChars != 2000 {
		t.Fatalf("unexpected disabled config: %v %v", c, err)
	}
	settings := Settings{Enabled: true, Provider: "QWEN", Qwen: ProviderSettings{Model: "fixture-qwen"}, DeepSeek: ProviderSettings{Model: "fixture-deepseek"}}
	c, err = loadConfig(settings, getenv)
	if err != nil || c.Provider != ProviderQwen || c.BaseURL != "https://dashscope.aliyuncs.com/compatible-mode/v1" || c.APIKey != "test-secret" {
		t.Fatalf("unexpected provider config: %v %v", c, err)
	}
	data, _ := json.Marshal(c)
	for _, formatted := range []string{fmt.Sprint(c), fmt.Sprintf("%+v", c), fmt.Sprintf("%#v", c), string(data)} {
		if strings.Contains(formatted, "test-secret") {
			t.Fatal("config exposed API key")
		}
	}
	delete(values, "QWEN_API_KEY")
	if _, err := loadConfig(settings, getenv); err == nil {
		t.Fatal("enabled agent accepted missing credentials")
	}
}

func TestConfigurationRejectsUnsafeEndpointsAndExcessiveBudgets(t *testing.T) {
	base := Config{Enabled: true, Provider: ProviderDeepSeek, Model: "deepseek-chat", APIKey: "test-key", BaseURL: "https://api.deepseek.com/v1"}
	for _, endpoint := range []string{"http://api.deepseek.com", "http://127.0.0.1:1234", "https://u:p@example.com", "https://example.com?key=x", "https://example.com/#secret", "file:///tmp/model"} {
		c := base
		c.BaseURL = endpoint
		if _, err := c.normalized(); err == nil {
			t.Fatalf("accepted unsafe endpoint %s", endpoint)
		}
	}
	c := base
	c.BaseURL = "http://127.0.0.1:1234"
	c.AllowHTTPForLoopback = true
	if _, err := c.normalized(); err != nil {
		t.Fatal("explicit loopback fixture should be supported")
	}
	c.BaseURL = "http://192.0.2.1:1234"
	if _, err := c.normalized(); err == nil {
		t.Fatal("loopback flag allowed a remote HTTP endpoint")
	}
	c = base
	c.MaxSteps = HardMaxSteps + 1
	if _, err := c.normalized(); err == nil {
		t.Fatal("accepted unbounded steps")
	}
}

func TestYAMLSettingsCannotEnableTestHTTP(t *testing.T) {
	settings := Settings{Enabled: true, Provider: ProviderDeepSeek, DeepSeek: ProviderSettings{Model: "fixture-model", BaseURL: "http://127.0.0.1:1234"}}
	_, err := loadConfig(settings, func(string) string { return "test-key" })
	if err == nil {
		t.Fatal("service settings allowed test-only HTTP")
	}
}

func TestChangingProviderDoesNotReuseAnotherProvidersEndpoint(t *testing.T) {
	values := map[string]string{"DEEPSEEK_API_KEY": "deepseek-key", "QWEN_API_KEY": "qwen-key"}
	settings := Settings{Enabled: true, Provider: ProviderDeepSeek, DeepSeek: ProviderSettings{Model: "fixture-deepseek", BaseURL: "https://api.deepseek.com/v1"}, Qwen: ProviderSettings{Model: "fixture-qwen"}}
	getenv := func(name string) string { return values[name] }
	c, err := loadConfig(settings, getenv)
	if err != nil || c.Model != "fixture-deepseek" || c.APIKey != "deepseek-key" {
		t.Fatal("DeepSeek settings were not selected", c, err)
	}
	settings.Provider = ProviderQwen
	c, err = loadConfig(settings, getenv)
	if err != nil || c.BaseURL != "https://dashscope.aliyuncs.com/compatible-mode/v1" || c.Model != "fixture-qwen" || c.APIKey != "qwen-key" {
		t.Fatalf("provider switch reused old secrets or endpoint: %v %v", c, err)
	}
}

func TestLegacyParameterEnvironmentDoesNotOverrideYAML(t *testing.T) {
	values := map[string]string{
		"AI_AGENT_ENABLED": "true", "AI_AGENT_PROVIDER": "qwen", "AI_AGENT_MAX_INPUT_CHARS": "8000",
		"AI_AGENT_MAX_STEPS": "8", "AI_AGENT_MAX_RUN_SECONDS": "180", "DEEPSEEK_MODEL": "old-model",
		"DEEPSEEK_BASE_URL": "https://old.example.com", "DEEPSEEK_API_KEY": "test-key", "QWEN_API_KEY": "other-key",
	}
	settings := Settings{Enabled: true, Provider: ProviderDeepSeek, MaxInputChars: 1234, MaxSteps: 3, MaxRunSeconds: 45, DeepSeek: ProviderSettings{Model: "yaml-model", BaseURL: "https://yaml.example.com/v1"}}
	c, err := loadConfig(settings, func(name string) string {
		if name != "DEEPSEEK_API_KEY" {
			t.Fatalf("configuration unexpectedly read %s", name)
		}
		return values[name]
	})
	if err != nil || !c.Enabled || c.Provider != ProviderDeepSeek || c.Model != "yaml-model" || c.BaseURL != "https://yaml.example.com/v1" || c.MaxInputChars != 1234 || c.MaxSteps != 3 || c.MaxRunSeconds != 45 {
		t.Fatal("environment overrode YAML parameters", c, err)
	}
	settings.Enabled = false
	c, err = loadConfig(settings, func(string) string {
		t.Fatal("legacy environment enabled a disabled feature")
		return ""
	})
	if err != nil || c.Enabled {
		t.Fatal("disabled YAML setting was ignored", c, err)
	}
}
