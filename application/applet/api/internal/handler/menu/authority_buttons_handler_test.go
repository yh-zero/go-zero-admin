package menu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"go-zero-admin/application/applet/api/internal/svc"
	menuclient "go-zero-admin/application/applet/rpc/client/menu"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type buttonRPCStub struct {
	menuclient.Menu
	failure     error
	authorityID int64
	ids         []int64
}

func (s *buttonRPCStub) GetAuthorityButtons(_ context.Context, in *pb.GetAuthorityButtonsRequest, _ ...grpc.CallOption) (*pb.GetAuthorityButtonsResponse, error) {
	s.authorityID = in.AuthorityId
	if s.failure != nil {
		return nil, s.failure
	}
	return &pb.GetAuthorityButtonsResponse{MenuBtnIds: []int64{7, 8}}, nil
}
func (s *buttonRPCStub) UpdateAuthorityButtons(_ context.Context, in *pb.UpdateAuthorityButtonsRequest, _ ...grpc.CallOption) (*pb.NoDataResponse, error) {
	s.authorityID = in.AuthorityId
	s.ids = in.MenuBtnIds
	if s.failure != nil {
		return nil, s.failure
	}
	return &pb.NoDataResponse{}, nil
}
func TestAuthorityButtonHTTPEnvelope(t *testing.T) {
	stub := &buttonRPCStub{}
	service := &svc.ServiceContext{AppletMenuRPC: stub}
	response := httptest.NewRecorder()
	GetAuthorityButtonsHandler(service)(response, httptest.NewRequest(http.MethodGet, "/v1/sys/menu/getAuthorityButtons?authorityId=42", nil))
	var read struct {
		Code   uint32 `json:"code"`
		Result struct {
			IDs []int64 `json:"menuBtnIds"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &read); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || read.Code != 200 || !reflect.DeepEqual(read.Result.IDs, []int64{7, 8}) || stub.authorityID != 42 {
		t.Fatalf("GET must wrap and preserve payload: %s", response.Body.String())
	}
	request := httptest.NewRequest(http.MethodPut, "/v1/sys/menu/updateAuthorityButtons", strings.NewReader(`{"authorityId":42,"menuBtnIds":[8]}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	UpdateAuthorityButtonsHandler(service)(response, request)
	var saved struct {
		Code   uint32 `json:"code"`
		Result struct {
			Message string `json:"message"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || saved.Code != 200 || saved.Result.Message == "" || stub.authorityID != 42 || !reflect.DeepEqual(stub.ids, []int64{8}) {
		t.Fatalf("PUT must wrap and pass request: %s", response.Body.String())
	}
}
func TestAuthorityButtonHTTPBusinessError(t *testing.T) {
	stub := &buttonRPCStub{failure: status.Convert(xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "角色不存在")).Err()}
	response := httptest.NewRecorder()
	GetAuthorityButtonsHandler(&svc.ServiceContext{AppletMenuRPC: stub})(response, httptest.NewRequest(http.MethodGet, "/v1/sys/menu/getAuthorityButtons?authorityId=99", nil))
	var output struct {
		Code    uint32 `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || output.Code != xerr.REUQEST_PARAM_ERROR || output.Message != "角色不存在" {
		t.Fatalf("business error envelope lost: %s", response.Body.String())
	}
}
