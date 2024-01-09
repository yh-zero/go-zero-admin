// Code generated by goctl. DO NOT EDIT.
// Source: applet.proto

package server

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/logic/casbin"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
)

type CasbinServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedCasbinServer
}

func NewCasbinServer(svcCtx *svc.ServiceContext) *CasbinServer {
	return &CasbinServer{
		svcCtx: svcCtx,
	}
}

// 根据角色id获取对应的casbin数据
func (s *CasbinServer) GetPathByAuthorityId(ctx context.Context, in *pb.GetPathByAuthorityIdRequest) (*pb.GetPathByAuthorityIdResponse, error) {
	l := casbinlogic.NewGetPathByAuthorityIdLogic(ctx, s.svcCtx)
	return l.GetPathByAuthorityId(in)
}

// 更新一个角色的对应的casbin数据
func (s *CasbinServer) UpdateCasbinData(ctx context.Context, in *pb.UpdateCasbinDataRequest) (*pb.NoDataResponse, error) {
	l := casbinlogic.NewUpdateCasbinDataLogic(ctx, s.svcCtx)
	return l.UpdateCasbinData(in)
}

// 更新一个角色的对应的casbin数据 用api的ids 查数据
func (s *CasbinServer) UpdateCasbinDataByApiIds(ctx context.Context, in *pb.UpdateCasbinDataByApiIdsRequest) (*pb.NoDataResponse, error) {
	l := casbinlogic.NewUpdateCasbinDataByApiIdsLogic(ctx, s.svcCtx)
	return l.UpdateCasbinDataByApiIds(in)
}
