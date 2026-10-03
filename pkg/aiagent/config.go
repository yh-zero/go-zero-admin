// Package aiagent runs the built-in read-only assistant. Business authorization
// belongs in the injected tool callbacks, and must be checked on every call.
package aiagent

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	ProviderDeepSeek = "deepseek"
	ProviderQwen     = "qwen"
	HardMaxSteps     = 8
)

// Settings contains the non-secret service YAML configuration. Model credentials
// remain in the selected provider's environment variable, never in this struct.
type Settings struct {
	Enabled       bool             `json:",optional"`
	Provider      string           `json:",default=deepseek"`
	MaxInputChars int              `json:",optional"`
	MaxSteps      int              `json:",optional"`
	MaxRunSeconds int              `json:",optional"`
	DeepSeek      ProviderSettings `json:",optional"`
	Qwen          ProviderSettings `json:",optional"`
}

type ProviderSettings struct {
	Model   string `json:",optional"`
	BaseURL string `json:",optional"`
}

// Config contains server-side settings only. APIKey is never serialized or
// included in formatted Config values. AllowHTTPForLoopback is for local tests;
// LoadConfig deliberately cannot enable it.
type Config struct {
	Enabled              bool
	Provider             string
	BaseURL              string
	Model                string
	APIKey               string `json:"-" yaml:"-"`
	AllowHTTPForLoopback bool
	MaxInputChars        int
	MaxRunSeconds        int
	RequestTimeout       time.Duration
	RunTimeout           time.Duration
	MaxSteps             int
	MaxOutputTokens      int
	MaxTotalTokens       int
	MaxAnswerBytes       int
	MaxToolOutputBytes   int
	MaxResponseBytes     int64
}

func (c Config) String() string {
	return fmt.Sprintf("aiagent.Config{Enabled:%t Provider:%q Model:%q APIKey:[redacted]}", c.Enabled, c.Provider, c.Model)
}

func (c Config) GoString() string { return c.String() }

// LoadConfig reads parameters from service YAML and only credentials from the
// environment. Legacy AI_AGENT_*, *_MODEL and *_BASE_URL variables do not override
// YAML. Disabled mode never reads model credentials or contacts a provider.
func LoadConfig(settings Settings) (Config, error) {
	return loadConfig(settings, os.Getenv)
}

func loadConfig(settings Settings, getenv func(string) string) (Config, error) {
	c := Config{
		Enabled: settings.Enabled, Provider: strings.ToLower(strings.TrimSpace(settings.Provider)),
		MaxInputChars: settings.MaxInputChars, MaxSteps: settings.MaxSteps, MaxRunSeconds: settings.MaxRunSeconds,
	}
	if c.Provider == "" {
		c.Provider = ProviderDeepSeek
	}
	keyName := ""
	switch c.Provider {
	case ProviderDeepSeek:
		c.Model, c.BaseURL = settings.DeepSeek.Model, settings.DeepSeek.BaseURL
		keyName = "DEEPSEEK_API_KEY"
	case ProviderQwen:
		c.Model, c.BaseURL = settings.Qwen.Model, settings.Qwen.BaseURL
		keyName = "QWEN_API_KEY"
	}
	if c.Enabled && keyName != "" {
		c.APIKey = getenv(keyName)
	}
	return c.normalized()
}

func (c Config) normalized() (Config, error) {
	c.Provider = strings.ToLower(strings.TrimSpace(c.Provider))
	c.Model = strings.TrimSpace(c.Model)
	c.APIKey = strings.TrimSpace(c.APIKey)
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if c.BaseURL == "" {
		switch c.Provider {
		case ProviderDeepSeek:
			c.BaseURL = "https://api.deepseek.com/v1"
		case ProviderQwen:
			c.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		}
	}
	if c.MaxInputChars == 0 {
		c.MaxInputChars = 2000
	}
	if c.MaxRunSeconds == 0 {
		c.MaxRunSeconds = 90
	}
	if c.RequestTimeout == 0 {
		c.RequestTimeout = 30 * time.Second
	}
	if c.RunTimeout == 0 {
		c.RunTimeout = time.Duration(c.MaxRunSeconds) * time.Second
	}
	if cap := time.Duration(c.MaxRunSeconds) * time.Second; c.RunTimeout > cap {
		c.RunTimeout = cap
	}
	if c.MaxSteps == 0 {
		c.MaxSteps = 6
	}
	if c.MaxOutputTokens == 0 {
		c.MaxOutputTokens = 1024
	}
	if c.MaxTotalTokens == 0 {
		c.MaxTotalTokens = 24000
	}
	if c.MaxAnswerBytes == 0 {
		c.MaxAnswerBytes = 8192
	}
	if c.MaxToolOutputBytes == 0 {
		c.MaxToolOutputBytes = 16384
	}
	if c.MaxResponseBytes == 0 {
		c.MaxResponseBytes = 1 << 20
	}
	if c.MaxInputChars < 1 || c.MaxInputChars > 8000 || c.MaxRunSeconds < 1 || c.MaxRunSeconds > 180 || c.RequestTimeout <= 0 || c.RequestTimeout > 60*time.Second || c.RunTimeout <= 0 || c.RunTimeout > 3*time.Minute || c.MaxSteps < 1 || c.MaxSteps > HardMaxSteps || c.MaxOutputTokens < 1 || c.MaxOutputTokens > 4096 || c.MaxTotalTokens < 1 || c.MaxTotalTokens > 100000 || c.MaxAnswerBytes < 1 || c.MaxAnswerBytes > 32768 || c.MaxToolOutputBytes < 1 || c.MaxToolOutputBytes > 32768 || c.MaxResponseBytes < 1 || c.MaxResponseBytes > 2<<20 {
		return c, fmt.Errorf("invalid AI agent timeout or budget")
	}
	if !c.Enabled {
		return c, nil
	}
	if c.Provider != ProviderDeepSeek && c.Provider != ProviderQwen {
		return c, fmt.Errorf("AI agent provider must be deepseek or qwen")
	}
	if c.APIKey == "" || c.Model == "" {
		return c, fmt.Errorf("AI agent API key and model are required")
	}
	if len(c.APIKey) > 4096 || strings.ContainsAny(c.APIKey, "\r\n") || len(c.Model) > 128 || strings.ContainsAny(c.Model, "\r\n\x00") {
		return c, fmt.Errorf("invalid AI agent key or model")
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return c, fmt.Errorf("invalid AI agent base URL")
	}
	if u.Scheme != "https" {
		ip := net.ParseIP(u.Hostname())
		if u.Scheme != "http" || !c.AllowHTTPForLoopback || ip == nil || !ip.IsLoopback() {
			return c, fmt.Errorf("AI agent base URL must use HTTPS")
		}
	}
	return c, nil
}
