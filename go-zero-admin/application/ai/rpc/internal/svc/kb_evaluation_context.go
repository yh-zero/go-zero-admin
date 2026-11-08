package svc

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"

	"go-zero-admin/application/ai/rpc/internal/config"
)

var evaluationSensitiveLabel = regexp.MustCompile(`(?i)Bearer\s|sk-[A-Za-z0-9_-]{12,}|eyJ[A-Za-z0-9_-]{12,}\.|[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}`)

func newEvaluationContext(ai config.ConfigAI) (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", errors.New("无法核对运行构建")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", errors.New("无法读取运行构建")
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", errors.New("无法计算运行构建摘要")
	}
	return evaluationContextForBuild(ai, hex.EncodeToString(digest.Sum(nil)))
}

func evaluationContextForBuild(ai config.ConfigAI, binarySHA string) (string, error) {
	invalid := func() (string, error) { return "", errors.New("质量评测配置缺失或不适合公开基线") }
	if len(binarySHA) != 64 || strings.Trim(binarySHA, "0123456789abcdef") != "" || ai.MaxContextTokens < 4 || ai.MaxContextTokens > 9007199254740991 || ai.EmbeddingDimensions <= 0 || ai.EmbeddingDimensions > 9007199254740991 || ai.TimeoutSeconds < 1 || ai.TimeoutSeconds > 90 {
		return invalid()
	}
	for _, label := range []string{ai.Provider, ai.Model, ai.EmbeddingProvider, ai.EmbeddingModel} {
		if strings.TrimSpace(label) == "" || evaluationSensitiveLabel.MatchString(label) {
			return invalid()
		}
	}
	for _, endpoint := range []string{ai.BaseURL, ai.EmbeddingBaseURL} {
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || evaluationSensitiveLabel.MatchString(endpoint) {
			return invalid()
		}
	}
	// 仅显式列出可公开的有效配置，禁止序列化包含密钥的ConfigAI。
	value := map[string]any{
		"binary_sha": binarySHA,
		"model":      map[string]any{"provider": strings.ToLower(strings.TrimSpace(ai.Provider)), "name": ai.Model, "base_url": ai.BaseURL},
		"embedding":  map[string]any{"provider": strings.ToLower(strings.TrimSpace(ai.EmbeddingProvider)), "model": ai.EmbeddingModel, "dimension": ai.EmbeddingDimensions, "base_url": ai.EmbeddingBaseURL},
		"config":     map[string]any{"max_context_tokens": ai.MaxContextTokens, "analysis_answer_token_cap": min(int64(2048), ai.MaxContextTokens/4), "recall_top_k": kbSearchRecallTopK, "return_top_k": kbSearchReturnTopK, "analysis_temperature": 0, "analysis_timeout_seconds": ai.TimeoutSeconds},
	}
	encoded, err := json.Marshal(value)
	return string(encoded), err
}
