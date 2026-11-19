package aiservicelogic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/agenttools"
	"google.golang.org/protobuf/proto"
)

type documentChatRecall struct {
	owner, kb int64
	hits      map[int64]agenttools.KbSearchHit
	queries   []string
}

func (s *documentChatRecall) Search(context.Context, int64, int64, string) ([]agenttools.KbSearchHit, error) {
	return nil, errors.New("文档对话调用了整库检索")
}

func (s *documentChatRecall) SearchDocument(_ context.Context, owner, kb, doc int64, query string) ([]agenttools.KbSearchHit, error) {
	s.queries = append(s.queries, query)
	hit, ok := s.hits[doc]
	if owner != s.owner || kb != s.kb || !ok {
		return nil, errors.New("检索收到无权访问的资料")
	}
	if query == "是否能报销宠物寄养？" {
		return nil, nil
	}
	return []agenttools.KbSearchHit{hit}, nil
}

func seedDocumentChatKB(t *testing.T, conn sqlx.SqlConn, owner int64) int64 {
	t.Helper()
	r, err := conn.ExecCtx(context.Background(), "INSERT INTO ai_kb(owner_id,name) VALUES (?,?)", owner, uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	kb, err := r.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := conn.ExecCtx(context.Background(), "DELETE FROM ai_kb_doc WHERE kb_id=?", kb); err != nil {
			t.Error(err)
		}
		if _, err := conn.ExecCtx(context.Background(), "DELETE FROM ai_kb WHERE id=? AND owner_id=?", kb, owner); err != nil {
			t.Error(err)
		}
	})
	return kb
}

func seedChatDocument(t *testing.T, conn sqlx.SqlConn, kb int64, title, content, status string, version int64) agenttools.KbSearchHit {
	t.Helper()
	r, err := conn.ExecCtx(context.Background(), "INSERT INTO ai_kb_doc(kb_id,title,content,status,version) VALUES (?,?,?,?,?)", kb, title, content, status, version)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := r.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return agenttools.KbSearchHit{KbID: kb, DocID: doc, DocTitle: title, DocVersion: version, ChunkID: doc + 1000, Snippet: content, Score: 0.9}
}

func TestChatDocumentScopeIsolationHistoryAndReplay(t *testing.T) {
	ctx := context.Background()
	var mu sync.Mutex
	var prompts []string
	modelPrompts := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), prompts...)
	}
	service, owner := aiIntegrationContext(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Tools    []json.RawMessage
			Messages []struct{ Role, Content string }
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Messages) != 2 || len(body.Tools) != 0 || body.Messages[0].Role != "system" || body.Messages[1].Role != "user" {
			t.Error("限定文档的模型调用应只有规则和本轮材料")
			http.Error(w, "请求格式异常", 400)
			return
		}
		prompt := body.Messages[1].Content
		mu.Lock()
		prompts = append(prompts, prompt)
		mu.Unlock()
		amount := "300元"
		if strings.Contains(prompt, "每晚800元") {
			amount = "800元"
		}
		if strings.Contains(prompt, "每晚300元") == strings.Contains(prompt, "每晚800元") {
			t.Error("模型材料应当只包含所选文档的额度")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":%q},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":2,\"completion_tokens\":3,\"total_tokens\":5}}\n\ndata: [DONE]\n\n", amount+" [1]")
	})
	cacheConfig := cache.CacheConf{{RedisConf: redis.RedisConf{Host: os.Getenv("GOZERO_AI_TEST_REDIS"), Type: "node"}, Weight: 100}}
	service.AiKbDocModel = model.NewAiKbDocModel(service.DB, cacheConfig)
	service.AiKbModel = model.NewAiKbModel(service.DB, cacheConfig)
	kb := seedDocumentChatKB(t, service.DB, owner)
	local := seedChatDocument(t, service.DB, kb, "本地住宿制度", "本地住宿每晚300元", "ready", 7)
	remote := seedChatDocument(t, service.DB, kb, "外派住宿制度", "外派住宿每晚800元", "ready", 2)
	recall := &documentChatRecall{owner: owner, kb: kb, hits: map[int64]agenttools.KbSearchHit{local.DocID: local, remote.DocID: remote}}
	service.KbSearcher = recall
	request := func(conversation, doc int64, question string) *aipb.SendMessageRequest {
		return &aipb.SendMessageRequest{Actor: &aipb.Actor{UserId: owner}, RequestId: uuid.NewString(), ConversationId: conversation, KbId: kb, DocId: doc, Grounded: true, Content: question}
	}
	expected := map[int64]*aipb.ChatMessage{}
	checkDone := func(frame *aipb.StreamChunk, doc int64, hit *agenttools.KbSearchHit) {
		t.Helper()
		if frame.Type != "done" || frame.GetResult().GetUserMessage() == nil || frame.GetResult().GetAiMessage() == nil {
			t.Fatalf("消息未完成 type=%s code=%d", frame.Type, frame.ErrorCode)
		}
		u, a := frame.Result.UserMessage, frame.Result.AiMessage
		for _, msg := range []*aipb.ChatMessage{u, a} {
			if msg.Id <= 0 || msg.DocId != doc || msg.KbId != kb || !msg.Grounded || msg.Status != "succeeded" {
				t.Fatalf("消息未保存本轮范围：%v", msg)
			}
			expected[msg.Id] = msg
		}
		if u.Role != "user" || a.Role != "assistant" || len(u.Sources) != 0 {
			t.Fatal("消息角色或用户消息引用异常")
		}
		if hit == nil {
			if len(a.Sources) != 0 || !strings.Contains(a.Content, "这篇资料中未检索到") || frame.Tokens != 0 {
				t.Fatal("缺少证据时应零消耗且明确告知当前资料无命中")
			}
			return
		}
		if len(a.Sources) != 1 {
			t.Fatalf("实际引用数 %d", len(a.Sources))
		}
		source := a.Sources[0]
		if source.KbId != kb || source.DocId != doc || source.DocVersion != hit.DocVersion || source.ChunkId != hit.ChunkID || source.Snippet != hit.Snippet || source.DocTitle != hit.DocTitle {
			t.Fatalf("消息引用未还原当次检索事实：%v", source)
		}
	}

	first := request(0, local.DocID, "住宿额度是多少？")
	done := runAITestStream(t, ctx, service, first, nil).last()
	checkDone(done, local.DocID, &local)
	conv := done.Result.ConversationId
	follow := request(conv, local.DocID, "第二种呢？")
	checkDone(runAITestStream(t, ctx, service, follow, nil).last(), local.DocID, &local)
	switchDoc := request(conv, remote.DocID, follow.Content)
	checkDone(runAITestStream(t, ctx, service, switchDoc, nil).last(), remote.DocID, &remote)
	p := modelPrompts()
	if len(recall.queries) != 3 || len(p) != 3 || recall.queries[0] != first.Content || recall.queries[2] != switchDoc.Content {
		t.Fatalf("首轮或切换文档时不应继承旧查询：%v", recall.queries)
	}
	for _, q := range []string{recall.queries[1], p[1]} {
		if !strings.Contains(q, first.Content) || !strings.Contains(q, "300元 [1]") || !strings.Contains(q, follow.Content) {
			t.Fatalf("同文档追问丢失指代上下文：%s", q)
		}
	}
	if strings.Contains(p[0], "上轮回答：") || !strings.Contains(p[2], remote.Snippet) || strings.Contains(p[2], "300元") || strings.Contains(p[2], first.Content) {
		t.Fatal("首轮或切换文档后的模型材料包含越界上下文")
	}
	missing := request(conv, remote.DocID, "是否能报销宠物寄养？")
	checkDone(runAITestStream(t, ctx, service, missing, nil).last(), remote.DocID, nil)
	if len(modelPrompts()) != 3 || len(recall.queries) != 4 {
		t.Fatal("资料无命中应只检索一次，且不调用模型")
	}

	lost := request(conv, remote.DocID, "外派额度是多少？")
	var committed *aipb.SendMessageResponse
	stream := &aiTestStream{ctx: ctx, onSend: func(f *aipb.StreamChunk) error {
		if f.Type == "done" {
			committed = proto.Clone(f.Result).(*aipb.SendMessageResponse)
			return io.ErrClosedPipe
		}
		return nil
	}}
	if err := NewSendMessageStreamLogic(ctx, service).SendMessageStream(lost, stream); !errors.Is(err, io.ErrClosedPipe) || committed == nil {
		t.Fatalf("提交后终帧写失败未正确传播：%v", err)
	}
	replay := runAITestStream(t, ctx, service, lost, nil).last()
	checkDone(replay, remote.DocID, &remote)
	if !proto.Equal(committed.UserMessage, replay.Result.UserMessage) || !proto.Equal(committed.AiMessage, replay.Result.AiMessage) {
		t.Fatal("断流重查返回了不同的已提交消息或资料版本")
	}
	changed := proto.Clone(lost).(*aipb.SendMessageRequest)
	changed.DocId = local.DocID
	frame := runAITestStream(t, ctx, service, changed, nil).last()
	if frame.Type != "error" || frame.ErrorCode != 400 || len(modelPrompts()) != 4 || len(recall.queries) != 5 {
		t.Fatal("断流重查或冲突请求重复生成，或允许同一请求改变文档")
	}
	read, err := NewGetConversationLogic(ctx, service).GetConversation(&aipb.GetConversationRequest{Actor: first.Actor, ConversationId: conv})
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Messages) != len(expected) {
		t.Fatalf("历史消息数=%d，预期=%d", len(read.Messages), len(expected))
	}
	for _, msg := range read.Messages {
		if !proto.Equal(msg, expected[msg.Id]) {
			t.Fatalf("历史与首发/重放消息不一致 id=%d", msg.Id)
		}
	}

	otherKB := seedDocumentChatKB(t, service.DB, owner)
	foreignKB := seedDocumentChatKB(t, service.DB, owner+1)
	foreign := seedChatDocument(t, service.DB, foreignKB, "他人资料", "他人制度", "ready", 1)
	pending := seedChatDocument(t, service.DB, kb, "处理中", "未就绪", "pending", 1)
	deleted := seedChatDocument(t, service.DB, kb, "已删除", "无效", "ready", 1)
	if _, err := service.DB.ExecCtx(ctx, "UPDATE ai_kb_doc SET deleted_at=NOW() WHERE id=? AND kb_id=?", deleted.DocID, kb); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		kb, doc  int64
		grounded bool
		code     int32
	}{
		{"缺少知识库", 0, local.DocID, true, 400}, {"非资料模式", kb, local.DocID, false, 400}, {"负文档编号", kb, -1, true, 400},
		{"资料不属于指定库", otherKB, local.DocID, true, 404}, {"其他用户的资料", foreignKB, foreign.DocID, true, 404},
		{"资料已删除", kb, deleted.DocID, true, 404}, {"资料未就绪", kb, pending.DocID, true, 409},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := request(conv, tc.doc, "额度是多少？")
			in.KbId, in.Grounded = tc.kb, tc.grounded
			f := runAITestStream(t, ctx, service, in, nil).last()
			if f.Type != "error" || f.ErrorCode != tc.code || len(modelPrompts()) != 4 || len(recall.queries) != 5 {
				t.Fatalf("范围无效时应在检索前拒绝，期望 %d，实际 type=%s code=%d", tc.code, f.Type, f.ErrorCode)
			}
		})
	}
}
