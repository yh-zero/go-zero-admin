package ai

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/client/aiservice"
	"google.golang.org/grpc"
)

type kbChoicesClient struct {
	aiservice.AiService
	rows []*aipb.KbChoice
}

func (c *kbChoicesClient) ListKbs(context.Context, *aipb.ListKbsRequest, ...grpc.CallOption) (*aipb.ListKbsResponse, error) {
	return &aipb.ListKbsResponse{Choices: c.rows}, nil
}

func TestListKbsChoicesJSON(t *testing.T) {
	ctx := context.WithValue(context.Background(), middleware.CtxKeyUserID, int64(9))
	for _, tc := range []struct {
		name        string
		choicesOnly bool
		rows        []*aipb.KbChoice
		want        []types.AiKbChoice
	}{
		{"无匹配项", true, nil, []types.AiKbChoice{}},
		{"返回选项", true, []*aipb.KbChoice{{Id: 7, Name: "制度"}}, []types.AiKbChoice{{Id: 7, Name: "制度"}}},
		{"完整模式", false, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logic := NewListKbsLogic(ctx, &svc.ServiceContext{Validator: svc.NewValidator(), AiRpc: &kbChoicesClient{rows: tc.rows}})
			out, err := logic.ListKbs(&types.AiListKbsReq{ChoicesOnly: tc.choicesOnly})
			if err != nil {
				t.Fatal(err)
			}
			payload, err := json.Marshal(out)
			if err != nil {
				t.Fatal(err)
			}
			var response struct {
				Choices []types.AiKbChoice `json:"choices"`
			}
			if err := json.Unmarshal(payload, &response); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(response.Choices, tc.want) {
				t.Fatalf("choices = %#v, want %#v", response.Choices, tc.want)
			}
		})
	}
}
