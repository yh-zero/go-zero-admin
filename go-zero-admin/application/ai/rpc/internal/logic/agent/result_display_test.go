package agentlogic

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/pb"
	appletpb "go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/agentjobs"
	"go-zero-admin/pkg/aiagent"
)

const displayFixture = `{"id":1,"name":"report.pdf","mime":"application/pdf","size":1024,"status":"active","visibility":"private","references":0,"explanation":"无引用","objectKey":"never-display-object-key","downloadUrl":"https://private.invalid/never-display-url"}`

func displayRunner(t *testing.T, change func(), executionError error) eventRunner {
	t.Helper()
	return func(ctx context.Context, request aiagent.Request, _ aiagent.EventHandler) (aiagent.Result, error) {
		for _, tool := range request.Tools {
			if tool.Name == "get_file_status" {
				out, err := tool.Call(ctx, json.RawMessage(`{"fileId":1}`))
				if err != nil {
					t.Error(err)
					return aiagent.Result{}, err
				}
				if change != nil {
					change()
				}
				return aiagent.Result{FinalAnswer: "已查询，但模型未列出记录。", Provider: "deepseek", Model: "fixture", Usage: aiagent.Usage{PromptTokens: 10, CompletionTokens: 5}, ToolSummaries: []aiagent.ToolSummary{{Name: tool.Name, Summary: out.Summary, Count: out.Count, Success: true}}}, executionError
			}
		}
		t.Error("authorized file tool missing")
		return aiagent.Result{}, aiagent.ErrToolFailed
	}
}

func installDisplayFixture(b *businessStub) {
	b.tool = func(context.Context, *appletpb.AgentToolRequest) (*appletpb.AgentToolResult, error) {
		return &appletpb.AgentToolResult{Content: displayFixture, Summary: "已查询文件", Count: 1}, nil
	}
}

func TestQueryDetailsReachRunAndHistoryWhenModelOmitsRecords(t *testing.T) {
	s, actor, business := fixture(t)
	installDisplayFixture(business)
	s.AgentRunner = displayRunner(t, nil, nil)
	if err := s.AgentJobs.Start(); err != nil {
		t.Fatal(err)
	}
	created, err := NewCreateAgentRunLogic(context.Background(), s).CreateAgentRun(&pb.CreateAgentRunRequest{Actor: actor, RequestId: "90000000-0000-4000-8000-000000000001", Message: "显示文件1的查询数据"})
	if err != nil {
		t.Fatal(err)
	}
	run := waitPipelineRun(t, s, actor.UserID, created.ID)
	if run.Status != agentjobs.StatusSucceeded || !strings.Contains(run.Text, `report\.pdf`) || !strings.Contains(run.Text, "查询明细") {
		t.Fatal("successful query lost its actual records", run.Status, run.Error)
	}
	if strings.Contains(run.Text, "never-display") || strings.Contains(run.ExtraJSON, "report.pdf") {
		t.Fatal("private fields or business contents leaked to progress")
	}
	response, err := NewGetAgentRunLogic(context.Background(), s).GetAgentRun(&pb.AgentIDRequest{Actor: actor, ID: created.ID})
	if err != nil || response.Answer != run.Text {
		t.Fatal("polling answer differs from persisted details", err)
	}
	history, err := NewListAgentMessagesLogic(context.Background(), s).ListAgentMessages(&pb.AgentMessageListRequest{Actor: actor, ID: created.ConversationId, PageNo: 1, PageSize: 20})
	if err != nil || history.Total != 2 || history.Items[0].Role != "assistant" || history.Items[0].Content != run.Text {
		t.Fatal("history lost query details", err)
	}
}

func TestQueryDetailsStayPrivateAfterFailureOrRevocation(t *testing.T) {
	for _, scenario := range []string{"model_failure", "tool_grant_revoked", "session_revoked", "canceled"} {
		t.Run(scenario, func(t *testing.T) {
			s, actor, business := fixture(t)
			installDisplayFixture(business)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var executionError error
			if scenario == "model_failure" {
				executionError = aiagent.ErrModelFailed
			}
			s.AgentRunner = displayRunner(t, func() {
				switch scenario {
				case "tool_grant_revoked":
					business.deny("/v1/sys/files/list", "GET", true)
				case "session_revoked":
					business.revoked.Store(true)
				case "canceled":
					cancel()
				}
			}, executionError)
			result, err := execute(ctx, s, agentjobs.Job{Run: agentjobs.Run{OwnerID: actor.UserID, AuthorityID: actor.AuthorityId, SessionID: actor.SessionID, SessionVersion: actor.SessionVersion, Question: "显示文件1"}})
			if err == nil || result.Text != "" || strings.Contains(result.ExtraJSON, "report.pdf") || result.InputTokens != 10 {
				t.Fatal("rejected execution disclosed details or lost known usage", err)
			}
		})
	}
}

func TestDisplayedDetailsFitAnswerBudgetWithoutTruncatingRows(t *testing.T) {
	tools, display := collectResultDisplay([]aiagent.Tool{{ToolDefinition: aiagent.ToolDefinition{Name: "get_file_status"}, Call: func(context.Context, json.RawMessage) (aiagent.ToolOutput, error) {
		return aiagent.ToolOutput{Content: displayFixture}, nil
	}}}, 8192)
	if _, err := tools[0].Call(context.Background(), json.RawMessage(`{"fileId":1}`)); err != nil {
		t.Fatal(err)
	}
	answer := display.answer(strings.Repeat("概", 2730), 8192)
	if !utf8.ValidString(answer) || len(answer) > 8192 || !strings.Contains(answer, `report\.pdf`) || !strings.Contains(answer, "已省略") {
		t.Fatal("display overflowed result budget or lost complete records")
	}
	_, none := collectResultDisplay(nil, 8192)
	if none.answer("普通回答", 8192) != "普通回答" {
		t.Fatal("non-query answer was changed")
	}
}
