package svc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/llm"
)

type analysisOwnedModel struct {
	model.AiKbModel
	calls int
	err   error
}

func (m *analysisOwnedModel) FindOwned(_ context.Context, id, owner int64) (*model.KbRow, error) {
	m.calls++
	if id != 2 || owner != 7 {
		return nil, errors.New("identity mismatch")
	}
	return &model.KbRow{Id: 2, OwnerId: 7}, m.err
}

func TestKbAnalyzerReusesActualSearcherAfterMySQLOwnership(t *testing.T) {
	var embeds, chats atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/embeddings" {
			embeds.Add(1)
			fmt.Fprint(w, `{"data":[{"index":0,"embedding":[1,0]}]}`)
			return
		}
		chats.Add(1)
		var request struct {
			MaxTokens int   `json:"max_tokens"`
			Tools     []any `json:"tools"`
		}
		if json.NewDecoder(r.Body).Decode(&request) != nil || request.MaxTokens != 2048 || len(request.Tools) > 0 {
			t.Error("分析模型须有界且无工具")
		}
		fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"依据资料分析。[1]"},"finish_reason":"stop"}],"usage":{"prompt_tokens":40,"completion_tokens":10,"total_tokens":50}}`)
	}))
	defer server.Close()
	embedder, err := llm.NewEmbedder(llm.EmbedConfig{BaseURL: server.URL, APIKey: "local-test", Model: "local-test", Dims: 2})
	if err != nil {
		t.Fatal(err)
	}
	client, err := llm.New(llm.Config{BaseURL: server.URL + "/v1", APIKey: "local-test", Model: "local-test", Timeout: 2})
	if err != nil {
		t.Fatal(err)
	}
	owned := &analysisOwnedModel{err: model.ErrNotFound}
	store, chunks := &recallStore{}, &recallChunks{}
	s := &ServiceContext{LLM: client, AiKbModel: owned, KbSearcher: &kbSearcher{store: store, embedder: embedder, chunks: chunks, snippetBudget: 8000}}
	tool := newKbAnalyzer(s)
	if tool == nil || !agenttools.IsWorkflowOnly(tool) {
		t.Fatal("配置齐全应装配工作流专用分析")
	}
	args := json.RawMessage(`{"kb_id":2,"query":"资料分析"}`)
	if _, err = tool.Execute(agenttools.WithActor(context.Background(), 7), args); err == nil || owned.calls != 1 || embeds.Load() != 0 || chats.Load() != 0 {
		t.Fatal("MySQL越权须在Embedding之前拒绝")
	}
	owned.err = nil
	out, err := tool.Execute(agenttools.WithActor(context.Background(), 7), args)
	if err != nil || owned.calls != 2 || embeds.Load() != 1 || chats.Load() != 1 {
		t.Fatalf("实际检索与模型各一次: err=%v owns=%d embeds=%d chats=%d", err, owned.calls, embeds.Load(), chats.Load())
	}
	var result agenttools.KbAnalysisResult
	if json.Unmarshal([]byte(out), &result) != nil || len(result.Sources) != 5 || result.Usage.TotalTokens != 50 || store.owner != 7 || store.kb != 2 || chunks.owner != 7 || chunks.kb != 2 {
		t.Fatal("结果或检索身份未复用实际实现")
	}
	s.LLM = nil
	if newKbAnalyzer(s) != nil {
		t.Fatal("聊天未配置不得注册分析")
	}
	s.LLM = client
	s.KbSearcher = nil
	if newKbAnalyzer(s) != nil {
		t.Fatal("检索未配置不得注册分析")
	}
}
