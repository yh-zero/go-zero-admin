package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/rest/pathvar"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/client/aiservice"
	"go-zero-admin/pkg/xerr"
	"google.golang.org/grpc"
)

type recallRouteClient struct {
	aiservice.AiService
	calls   int
	failure error
	empty   bool
}

func (c *recallRouteClient) SearchKb(_ context.Context, req *aipb.SearchKbRequest, _ ...grpc.CallOption) (*aipb.SearchKbResponse, error) {
	c.calls++
	if req.Actor.UserId != 7 || req.KbId != 2 || req.Query != "资料" {
		panic("route lost identity, path or query")
	}
	if c.failure != nil {
		return nil, c.failure
	}
	out := &aipb.SearchKbResponse{ElapsedMs: 42}
	if !c.empty {
		out.List = []*aipb.KbSearchHit{{ChunkId: 9, KbId: 2, DocId: 5, DocTitle: "标题", Snippet: "<script>纯文本</script>", Score: .8}}
	}
	return out, nil
}

func TestRegisteredKbSearchUsesJSONEnvelope(t *testing.T) {
	installRouteResponseHandlers(t)
	for _, tc := range []struct {
		name, body, id string
		actor          bool
		failure        error
		empty          bool
		code           uint32
		calls          int
	}{
		{name: "success", body: `{"query":"资料"}`, id: "2", actor: true, code: 200, calls: 1},
		{name: "empty", body: `{"query":"资料"}`, id: "2", actor: true, empty: true, code: 200, calls: 1},
		{name: "bad_json", body: `{`, id: "2", actor: true, code: 400},
		{name: "blank", body: `{"query":"　 \t"}`, id: "2", actor: true, code: 400},
		{name: "bad_id", body: `{"query":"资料"}`, id: "0", actor: true, code: 400},
		{name: "missing_actor", body: `{"query":"资料"}`, id: "2", code: 403},
		{name: "foreign", body: `{"query":"资料"}`, id: "2", actor: true, failure: xerr.NewErrCode(404), code: 404, calls: 1},
		{name: "unavailable", body: `{"query":"资料"}`, id: "2", actor: true, failure: xerr.NewErrCode(503), code: 503, calls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &recallRouteClient{failure: tc.failure, empty: tc.empty}
			route := registeredAIRoute(t, client, http.MethodPost, "/v1/sys/ai/kbs/:id/search")
			request := httptest.NewRequest(http.MethodPost, "/v1/sys/ai/kbs/"+tc.id+"/search", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept", "text/event-stream")
			request = pathvar.WithVars(request, map[string]string{"id": tc.id})
			if tc.actor {
				request = withRouteIdentity(request)
			}
			recorder := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
			route.Handler(recorder, request)
			assertJSONEnvelope(t, recorder.ResponseRecorder, tc.code)
			if client.calls != tc.calls || recorder.Flushed || len(recorder.deadlines) != 0 {
				t.Fatal("JSON route unexpectedly called RPC or enabled SSE")
			}
			if tc.code == 200 {
				var response struct {
					Result struct {
						List      []aipb.KbSearchHit `json:"list"`
						ElapsedMs int64              `json:"elapsed_ms"`
					} `json:"result"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Result.List == nil || response.Result.ElapsedMs != 42 {
					t.Fatalf("JSON shape lost: %s", recorder.Body.String())
				}
				if !tc.empty && (len(response.Result.List) != 1 || response.Result.List[0].ChunkId != 9 || response.Result.List[0].Snippet != "<script>纯文本</script>") {
					t.Fatal("hit content lost")
				}
			}
		})
	}
}
