package aiservicelogic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/agenttools"
	"google.golang.org/protobuf/proto"
)

type groundedOwnedKb struct {
	model.AiKbModel
	owner int64
}

func (m *groundedOwnedKb) FindOwned(_ context.Context, kb, owner int64) (*model.KbRow, error) {
	if kb != 2 || owner != m.owner {
		return nil, model.ErrNotFound
	}
	return &model.KbRow{Id: kb, OwnerId: owner}, nil
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
