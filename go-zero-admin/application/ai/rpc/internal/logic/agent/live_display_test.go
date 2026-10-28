package agentlogic

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/conf"
	"go-zero-admin/application/ai/rpc/pb"
	appletpb "go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/agentjobs"
	"go-zero-admin/pkg/aiagent"
)

// Explicitly opt in: this test makes at most two paid requests to the configured
// provider. All business rows, actors and RPC services are isolated fixtures.
func TestLiveProviderPersistsAuthorizedQueryDetails(t *testing.T) {
	if os.Getenv("GO_ZERO_AI_LIVE_TEST") != "1" {
		t.Skip("set GO_ZERO_AI_LIVE_TEST=1 and the selected provider key to run the synthetic live-model test")
	}
	var local struct{ AI aiagent.Settings }
	if conf.Load("../../../etc/ai.yaml", &local) != nil {
		t.Fatal("local AI configuration could not be loaded")
	}
	local.AI.Enabled = true
	cfg, err := aiagent.LoadConfig(local.AI)
	if err != nil {
		t.Fatal("selected model configuration or key is invalid")
	}
	cfg.MaxSteps, cfg.MaxOutputTokens, cfg.MaxTotalTokens = 2, 256, 8000
	cfg.RequestTimeout, cfg.RunTimeout = 30*time.Second, 45*time.Second
	s, actor, business := fixture(t)
	business.tool = func(_ context.Context, request *appletpb.AgentToolRequest) (*appletpb.AgentToolResult, error) {
		if request.ArgumentsJson == "" {
			t.Error("tool arguments missing")
		}
		return &appletpb.AgentToolResult{Content: `{"startTime":"2026-10-04T00:00:00+08:00","endTime":"2026-10-04T01:00:00+08:00","total":2,"byModule":[{"module":"test_audit","count":2}],"recent":[{"id":901,"actorName":"样例用户甲","module":"test_audit","action":"样例读取甲","result":"success","statusCode":200,"createdAt":"2026-10-04T00:10:00+08:00"},{"id":902,"actorName":"样例用户乙","module":"test_audit","action":"样例读取乙","result":"success","statusCode":200,"createdAt":"2026-10-04T00:20:00+08:00"}]}`, Summary: "匹配2条样例审计日志", Count: 2}, nil
	}
	s.AgentConfig = cfg
	s.AgentRunner, err = aiagent.New(context.Background(), cfg)
	if err != nil {
		t.Fatal("live model adapter initialization failed")
	}
	// The real async executor, authorization, business RPC and message persistence
	// remain in the path. Only the business rows are synthetic.
	s.AgentJobs.Close()
	s.AgentJobs, err = agentjobs.New(s.DB.DB, agentjobs.Config{Enabled: true, PollInterval: 5 * time.Millisecond, RunTimeout: 50 * time.Second}, NewAgentExecutor(s))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.AgentJobs.Close)
	if err := s.AgentJobs.Start(); err != nil {
		t.Fatal(err)
	}
	created, err := NewCreateAgentRunLogic(context.Background(), s).CreateAgentRun(&pb.CreateAgentRunRequest{Actor: actor, RequestId: "90000000-0000-4000-8000-000000000002", Message: "使用query_audit_logs查询这组样例审计数据，参数使用空对象。查询后只用一句话概述总数，不重复全部记录。"})
	if err != nil {
		t.Fatal("live synthetic task could not be submitted")
	}
	deadline := time.Now().Add(55 * time.Second)
	var run agentjobs.Run
	for time.Now().Before(deadline) {
		run, err = s.AgentJobs.Get(context.Background(), actor.UserID, created.ID)
		if err != nil {
			t.Fatal("synthetic task could not be read")
		}
		if agentjobs.IsTerminal(run.Status) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if run.Status != agentjobs.StatusSucceeded || business.toolCalls.Load() != 1 || !strings.Contains(run.Text, "样例用户甲") || !strings.Contains(run.Text, "样例用户乙") || !strings.Contains(run.Text, "901") || !strings.Contains(run.Text, "902") {
		t.Fatalf("live display failed: status=%s, businessCalls=%d, safeError=%s", run.Status, business.toolCalls.Load(), run.Error)
	}
	history, err := NewListAgentMessagesLogic(context.Background(), s).ListAgentMessages(&pb.AgentMessageListRequest{Actor: actor, ID: created.ConversationId, PageNo: 1, PageSize: 20})
	if err != nil || history.Total != 2 || history.Items[0].Content != run.Text {
		t.Fatal("live query details did not reach the persisted assistant message")
	}
	t.Logf("PASS provider=%s model=%s businessToolCalls=1 recordsDisplayed=2 inputTokens=%d outputTokens=%d", cfg.Provider, cfg.Model, run.InputTokens, run.OutputTokens)
}
