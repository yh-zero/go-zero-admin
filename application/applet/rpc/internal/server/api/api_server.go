// Code generated by goctl. DO NOT EDIT.
// Source: applet.proto

package server

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/logic/api"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
)

type ApiServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedApiServer
}

func NewApiServer(svcCtx *svc.ServiceContext) *ApiServer {
	return &ApiServer{
		svcCtx: svcCtx,
	}
}

// 获取API列表
func (s *ApiServer) GetApiList(ctx context.Context, in *pb.GetApiListRequest) (*pb.GetApiListResponse, error) {
	l := apilogic.NewGetApiListLogic(ctx, s.svcCtx)
	return l.GetApiList(in)
}

// 创建/添加 API列表
func (s *ApiServer) CreateApi(ctx context.Context, in *pb.CreateApiRequest) (*pb.NoDataResponse, error) {
	l := apilogic.NewCreateApiLogic(ctx, s.svcCtx)
	return l.CreateApi(in)
}

// 删除API列表
func (s *ApiServer) DeleteApi(ctx context.Context, in *pb.DeleteApiRequest) (*pb.NoDataResponse, error) {
	l := apilogic.NewDeleteApiLogic(ctx, s.svcCtx)
	return l.DeleteApi(in)
}

// 获取全部API列表
func (s *ApiServer) GetAllApiList(ctx context.Context, in *pb.NoDataResponse) (*pb.GetAllApiListResponse, error) {
	l := apilogic.NewGetAllApiListLogic(ctx, s.svcCtx)
	return l.GetAllApiList(in)
}

// 删除多条api
func (s *ApiServer) DeleteApisByIds(ctx context.Context, in *pb.DeleteApisByIdsRequest) (*pb.NoDataResponse, error) {
	l := apilogic.NewDeleteApisByIdsLogic(ctx, s.svcCtx)
	return l.DeleteApisByIds(in)
}

// 更新api
func (s *ApiServer) UpdateApi(ctx context.Context, in *pb.UpdateApiRequest) (*pb.NoDataResponse, error) {
	l := apilogic.NewUpdateApiLogic(ctx, s.svcCtx)
	return l.UpdateApi(in)
}
