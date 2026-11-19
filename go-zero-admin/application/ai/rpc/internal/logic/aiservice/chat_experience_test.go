package aiservicelogic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/xerr"
	"google.golang.org/protobuf/proto"
)

type groundedOwnedKb struct {
	model.AiKbModel
	owner int64
}

func TestChatMessageScopeKeepsHistoricalFingerprints(t *testing.T) {
	for _, tc := range []struct {
		kb       int64
		grounded bool
		hash     string
	}{
		{0, false, "23adabf0d25d4b010a5b282c8deac51eea9172383df42e7d14b7f57be33945f0"},
		{3, true, "f72dd76d856852d461e87d2a2e5b5738a319a58c6386dee524ff5b9afc399462"},
		{3, false, "896233fe0ecf2f238bb9dfa2cae4d07570bee5bae0d4cba4b834bc81469ec394"},
	} {
		if requestFingerprint(5, "hello", tc.kb, tc.grounded, 0) != tc.hash {
			t.Fatal("兼容已有请求时不应把默认文档 ID 纳入指纹")
		}
	}
	for _, grounded := range []bool{false, true} {
		msg := messageProto(&model.MessageRow{Role: "assistant", ToolCalls: callRecord{Version: 1, KbID: 3, Grounded: &grounded}.marshal()})
		if msg.KbId != 3 || msg.DocId != 0 || msg.Grounded != grounded {
			t.Fatal("重新打开会话时应还原用户选定的库和回答模式")
		}
	}
	legacy := messageProto(&model.MessageRow{Role: "assistant", ToolCalls: `{"version":1,"kb_id":3}`})
	if legacy.KbId != 0 || legacy.DocId != 0 || legacy.Grounded {
		t.Fatal("旧数据没有保存模式时应保持原兼容行为")
	}
}

func TestGroundedReferenceDistinguishesNewTopicAndFollowup(t *testing.T) {
	for _, tc := range []struct {
		question string
		followup bool
	}{
		{"服务器的部署端口是多少呢？", false}, {"第二季度的营业额是多少？", false},
		{"继续教育学分怎么申请？", false}, {"What are the IT department opening hours?", false},
		{"第二种呢？", true}, {"请继续。", true}, {"请继续讲解。", true},
		{"继续回答", true}, {"请继续回答。", true}, {"请继续回答吧！", true},
		{"What about the latter?", true}, {"How do I use it, exactly?", true}, {"Tell me more!", true},
	} {
		if got := isGroundedFollowup(tc.question); got != tc.followup {
			t.Errorf("%q 追问识别结果 %t，预期 %t", tc.question, got, tc.followup)
		}
	}
}

func (m *groundedOwnedKb) RequireOwned(_ context.Context, kb, owner int64) error {
	if (kb != 2 && kb != 3) || owner != m.owner {
		return model.ErrNotFound
	}
	return nil
}

func TestChatExperienceGroundingReplayAndFeedback(t *testing.T) {
	var calls atomic.Int32
	service, owner := aiIntegrationContext(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var payload struct {
			Tools    []json.RawMessage `json:"tools"`
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if json.NewDecoder(r.Body).Decode(&payload) != nil || len(payload.Tools) != 0 {
			t.Error("资料回答不得允许模型调用其他工具")
		}
		if len(payload.Messages) < 2 || !strings.Contains(payload.Messages[len(payload.Messages)-1].Content, "实际检索证据") {
			t.Error("模型未收到实际检索摘录")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"依据资料 [1]。\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":2,\"completion_tokens\":3,\"total_tokens\":5}}\n\ndata: [DONE]\n\n")
	})
	service.AiKbModel = &groundedOwnedKb{owner: owner}
	searcher := &directRecallSearcher{search: func(_ context.Context, gotOwner, kb int64, query string) ([]agenttools.KbSearchHit, error) {
		if gotOwner != owner || kb != 2 || query != "本次问题" {
			t.Error("查询身份或知识库范围不符")
		}
		return []agenttools.KbSearchHit{{ChunkID: 21, KbID: 2, DocID: 9, DocTitle: "<script>资料</script>", Snippet: "实际检索证据", Score: 0.8}}, nil
	}}
	service.KbSearcher = searcher
	req := &aipb.SendMessageRequest{Actor: &aipb.Actor{UserId: owner}, RequestId: uuid.NewString(), Content: "本次问题", KbId: 2, Grounded: true}
	result := runAITestStream(t, context.Background(), service, req, nil).last()
	if result.Type != "done" || calls.Load() != 1 || searcher.calls != 1 || len(result.GetResult().GetAiMessage().GetSources()) != 1 {
		t.Fatalf("资料回答或来源未完成: %s model=%d search=%d", result.Type, calls.Load(), searcher.calls)
	}
	conversation := result.Result.ConversationId
	message := result.Result.AiMessage.Id
	feedback := &aipb.SetMessageFeedbackRequest{Actor: req.Actor, ConversationId: conversation, MessageId: message, Feedback: "irrelevant_source", Note: "需要新版资料"}
	if _, err := NewSetMessageFeedbackLogic(context.Background(), service).SetMessageFeedback(feedback); err != nil {
		t.Fatal(err)
	}
	read, err := NewGetConversationLogic(context.Background(), service).GetConversation(&aipb.GetConversationRequest{Actor: req.Actor, ConversationId: conversation})
	if err != nil {
		t.Fatal(err)
	}
	var assistant *aipb.ChatMessage
	for _, row := range read.Messages {
		if row.Id == message {
			assistant = row
		}
	}
	if assistant == nil || assistant.Feedback != feedback.Feedback || assistant.FeedbackNote != feedback.Note || len(assistant.Sources) != 1 || assistant.Sources[0].Snippet != "实际检索证据" {
		t.Fatal("反馈覆盖真实来源或未持久化")
	}
	replay := runAITestStream(t, context.Background(), service, req, nil).last()
	if replay.Type != "done" || calls.Load() != 1 || searcher.calls != 1 || replay.Result.AiMessage.Feedback != feedback.Feedback {
		t.Fatal("重查重新执行或丢失反馈")
	}
	changed := proto.Clone(req).(*aipb.SendMessageRequest)
	changed.Grounded = false
	if frame := runAITestStream(t, context.Background(), service, changed, nil).last(); frame.Type != "error" || frame.ErrorCode != 400 {
		t.Fatal("不同资料模式复用了同一个请求ID")
	}
	feedback.Actor = &aipb.Actor{UserId: owner + 1}
	_, err = NewSetMessageFeedbackLogic(context.Background(), service).SetMessageFeedback(feedback)
	assertSearchCode(t, err, 404)
	feedback.Actor = req.Actor
	feedback.MessageId = result.Result.UserMessage.Id
	_, err = NewSetMessageFeedbackLogic(context.Background(), service).SetMessageFeedback(feedback)
	assertSearchCode(t, err, 404)
	feedback.MessageId = message
	feedback.Feedback = "__proto__"
	_, err = NewSetMessageFeedbackLogic(context.Background(), service).SetMessageFeedback(feedback)
	assertSearchCode(t, err, 400)
	feedback.Feedback = "helpful"
	feedback.Note = strings.Repeat("中", 334)
	_, err = NewSetMessageFeedbackLogic(context.Background(), service).SetMessageFeedback(feedback)
	assertSearchCode(t, err, 400)
	feedback.Feedback, feedback.Note = "", "清除时忽略此说明"
	if _, err := NewSetMessageFeedbackLogic(context.Background(), service).SetMessageFeedback(feedback); err != nil {
		t.Fatal(err)
	}
	replay = runAITestStream(t, context.Background(), service, req, nil).last()
	if replay.Result.AiMessage.Feedback != "" || replay.Result.AiMessage.FeedbackNote != "" {
		t.Fatal("反馈未清除")
	}
}

func TestChatExperienceGroundingNoMatchesAndOwnership(t *testing.T) {
	var calls atomic.Int32
	service, owner := aiIntegrationContext(t, func(http.ResponseWriter, *http.Request) { calls.Add(1); t.Error("无资料不应调用聊天模型") })
	service.AiKbModel = &groundedOwnedKb{owner: owner}
	searcher := &directRecallSearcher{search: func(context.Context, int64, int64, string) ([]agenttools.KbSearchHit, error) { return nil, nil }}
	service.KbSearcher = searcher
	req := &aipb.SendMessageRequest{Actor: &aipb.Actor{UserId: owner}, RequestId: uuid.NewString(), Content: "问题", KbId: 2, Grounded: true}
	result := runAITestStream(t, context.Background(), service, req, nil).last()
	if result.Type != "done" || calls.Load() != 0 || searcher.calls != 1 || result.Tokens != 0 || len(result.Result.AiMessage.Sources) != 0 || !strings.Contains(result.Result.AiMessage.Content, "未检索到") {
		t.Fatalf("无资料必须确定拒答且零聊天调用：type=%s code=%d tokens=%d search=%d chat=%d sources=%d", result.Type, result.ErrorCode, result.Tokens, searcher.calls, calls.Load(), len(result.GetResult().GetAiMessage().GetSources()))
	}
	req.RequestId, req.KbId = uuid.NewString(), 99
	result = runAITestStream(t, context.Background(), service, req, nil).last()
	if result.Type != "error" || result.ErrorCode != 404 || searcher.calls != 1 || calls.Load() != 0 {
		t.Fatal("知识库归属须在检索或模型之前校验")
	}
}

func TestChatSearchQuotaAndReplay(t *testing.T) {
	var modelCalls atomic.Int32
	service, owner := aiIntegrationContext(t, func(w http.ResponseWriter, r *http.Request) {
		modelCalls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		args := map[string]any{"name": "kb_search", "arguments": `{"query":"quota"}`}
		delta := map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "search", "type": "function", "function": args}}}
		writeModelFrame(t, w, map[string]any{"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": "tool_calls"}}})
		fmt.Fprint(w, "data: [DONE]\n\n")
	})
	service.AiKbModel = &groundedOwnedKb{owner: owner}
	const quotaMessage = "每日供应商请求额度已用完"
	searcher := &directRecallSearcher{search: func(context.Context, int64, int64, string) ([]agenttools.KbSearchHit, error) {
		return nil, fmt.Errorf("检索未准入: %w", xerr.NewErrCodeMsg(429, quotaMessage))
	}}
	service.KbSearcher = searcher
	agenttools.RegisterIfAbsent(agenttools.NewKbSearch(&budgetKbSearcher{}))
	for _, tc := range []struct {
		name           string
		grounded       bool
		modelCalls     int32
		groundedSearch int
	}{{"严格资料先检索", true, 0, 1}, {"普通聊天调用检索工具", false, 1, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			modelBefore, searchBefore := modelCalls.Load(), searcher.calls
			req := &aipb.SendMessageRequest{Actor: &aipb.Actor{UserId: owner}, RequestId: uuid.NewString(), Content: "问题", KbId: 2, Grounded: tc.grounded}
			for i := 0; i < 2; i++ {
				frame := runAITestStream(t, context.Background(), service, req, nil).last()
				if frame.Type != "error" || frame.ErrorCode != 429 || frame.Content != quotaMessage || modelCalls.Load()-modelBefore != tc.modelCalls || searcher.calls-searchBefore != tc.groundedSearch {
					t.Fatalf("第 %d 次读取错误帧=%v，聊天调用=%d，严格资料检索=%d", i+1, frame, modelCalls.Load(), searcher.calls)
				}
			}
		})
	}
}

func TestChatExperienceFollowupUsesFreshEvidenceWithinSelectedKB(t *testing.T) {
	var calls atomic.Int32
	service, owner := aiIntegrationContext(t, func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		var payload struct{ Messages []struct{ Content string } }
		if json.NewDecoder(r.Body).Decode(&payload) != nil || len(payload.Messages) != 2 {
			t.Error("严格模式只能使用本轮资料与有界的指代上下文")
			return
		}
		if n == 2 && (!strings.Contains(payload.Messages[0].Content, "不能作为证据") || !strings.Contains(payload.Messages[1].Content, "本轮复核资料")) {
			t.Error("追问缺少本轮证据或指代约束")
		}
		if n == 3 && strings.Contains(payload.Messages[1].Content, "上轮回答：") {
			t.Error("独立新问题不应携带前一轮问答")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"第二种是定期导入 [1]。\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":2,\"completion_tokens\":3,\"total_tokens\":5}}\n\ndata: [DONE]\n\n")
	})
	service.AiKbModel = &groundedOwnedKb{owner: owner}
	queries := []string{}
	service.KbSearcher = &directRecallSearcher{search: func(_ context.Context, gotOwner, kb int64, query string) ([]agenttools.KbSearchHit, error) {
		queries = append(queries, query)
		if gotOwner != owner {
			t.Fatal("检索身份变化")
		}
		if len(queries) == 5 {
			return nil, errors.New("fixture search error")
		}
		if kb == 3 {
			return nil, nil
		}
		id := int64(len(queries))
		return []agenttools.KbSearchHit{{KbID: kb, DocID: 5, ChunkID: id, Snippet: "本轮复核资料"}}, nil
	}}
	req := &aipb.SendMessageRequest{Actor: &aipb.Actor{UserId: owner}, RequestId: uuid.NewString(), Content: "导入有哪些方式？", KbId: 2, Grounded: true}
	first := runAITestStream(t, context.Background(), service, req, nil).last()
	if first.Type != "done" {
		t.Fatalf("首轮失败 %s/%d", first.Type, first.ErrorCode)
	}
	req.RequestId, req.ConversationId, req.Content = uuid.NewString(), first.Result.ConversationId, "第二种呢？"
	second := runAITestStream(t, context.Background(), service, req, nil).last()
	if second.Type != "done" || len(queries) != 2 || !strings.HasPrefix(queries[1], req.Content) || !strings.Contains(queries[1], "定期导入") || len(second.Result.AiMessage.Sources) != 1 || second.Result.AiMessage.Sources[0].ChunkId != 2 {
		t.Fatal("追问须重新检索并仅保存本轮证据")
	}
	req.RequestId, req.Content = uuid.NewString(), "继续教育学分怎么申请？"
	newTopic := runAITestStream(t, context.Background(), service, req, nil).last()
	if newTopic.Type != "done" || len(queries) != 3 || queries[2] != req.Content {
		t.Fatal("同一库的新主题应只检索当前问题")
	}
	req.RequestId, req.KbId, req.Content = uuid.NewString(), 3, "How do I use it, exactly?"
	empty := runAITestStream(t, context.Background(), service, req, nil).last()
	if empty.Type != "done" || len(queries) != 4 || queries[3] != req.Content || len(empty.Result.AiMessage.Sources) != 0 || calls.Load() != 3 {
		t.Fatal("切库后不得沿用旧问答或为无依据问题调用模型")
	}
	req.RequestId = uuid.NewString()
	failed := runAITestStream(t, context.Background(), service, req, nil).last()
	if failed.Type != "error" || failed.ErrorCode != 503 || calls.Load() != 3 {
		t.Fatal("检索失败须区别于无命中且不得调用模型")
	}
}

func TestChatExperienceConversationOrganization(t *testing.T) {
	service, owner := aiIntegrationContext(t, func(http.ResponseWriter, *http.Request) {})
	actor := &aipb.Actor{UserId: owner}
	created, err := NewCreateConversationLogic(context.Background(), service).CreateConversation(&aipb.CreateConversationRequest{Actor: actor, Title: "原始名称"})
	if err != nil {
		t.Fatal(err)
	}
	id := created.Conversation.Id
	other, err := NewCreateConversationLogic(context.Background(), service).CreateConversation(&aipb.CreateConversationRequest{Actor: actor, Title: "第二个会话"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.AiConversationModel.AppendTurn(context.Background(), other.Conversation.Id, owner, "独立正文关键字", "答复", "[]", 0); err != nil {
		t.Fatal(err)
	}
	title, pinned := "精确_%关键词", true
	updated, err := NewUpdateConversationLogic(context.Background(), service).UpdateConversation(&aipb.UpdateConversationRequest{Actor: actor, ConversationId: id, Title: &title, Pinned: &pinned})
	if err != nil || updated.Conversation.Title != title || !updated.Conversation.Pinned {
		t.Fatal("整理会话未保存")
	}
	for _, query := range []string{"关键词", "_%"} {
		list, err := NewListConversationsLogic(context.Background(), service).ListConversations(&aipb.ListConversationsRequest{Actor: actor, Query: query})
		if err != nil || len(list.List) != 1 || list.List[0].Id != id {
			t.Fatal("标题搜索或LIKE转义失败")
		}
	}
	body, err := NewListConversationsLogic(context.Background(), service).ListConversations(&aipb.ListConversationsRequest{Actor: actor, Query: "独立正文关键字"})
	if err != nil || len(body.List) != 1 || body.List[0].Id != other.Conversation.Id {
		t.Fatal("正文搜索失败")
	}
	all, err := NewListConversationsLogic(context.Background(), service).ListConversations(&aipb.ListConversationsRequest{Actor: actor})
	if err != nil || len(all.List) != 2 || all.List[0].Id != id {
		t.Fatal("置顶未优先排序")
	}
	foreign, err := NewListConversationsLogic(context.Background(), service).ListConversations(&aipb.ListConversationsRequest{Actor: &aipb.Actor{UserId: owner + 1}, Query: "独立正文关键字"})
	if err != nil || len(foreign.List) != 0 {
		t.Fatal("搜索泄露其他用户会话")
	}
	_, err = NewUpdateConversationLogic(context.Background(), service).UpdateConversation(&aipb.UpdateConversationRequest{Actor: &aipb.Actor{UserId: owner + 1}, ConversationId: id, Pinned: &pinned})
	assertSearchCode(t, err, 404)
	if _, err := NewDeleteConversationLogic(context.Background(), service).DeleteConversation(&aipb.DeleteConversationRequest{Actor: actor, ConversationId: id}); err != nil {
		t.Fatal(err)
	}
	_, err = NewUpdateConversationLogic(context.Background(), service).UpdateConversation(&aipb.UpdateConversationRequest{Actor: actor, ConversationId: id, Pinned: &pinned})
	assertSearchCode(t, err, 404)
}
