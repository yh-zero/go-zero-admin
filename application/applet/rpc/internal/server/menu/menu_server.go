// Code generated by goctl. DO NOT EDIT.
// Source: applet.proto

package server

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/logic/menu"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
)

type MenuServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedMenuServer
}

func NewMenuServer(svcCtx *svc.ServiceContext) *MenuServer {
	return &MenuServer{
		svcCtx: svcCtx,
	}
}

// 获取菜单-路由
func (s *MenuServer) GetMenuTree(ctx context.Context, in *pb.GetMenuTreeRequest) (*pb.GetMenuTreeResponse, error) {
	l := menulogic.NewGetMenuTreeLogic(ctx, s.svcCtx)
	return l.GetMenuTree(in)
}

// 获取系统基础菜单列表
func (s *MenuServer) GetMenuBaseInfoList(ctx context.Context, in *pb.NoDataResponse) (*pb.GetMenuBaseInfoListResponse, error) {
	l := menulogic.NewGetMenuBaseInfoListLogic(ctx, s.svcCtx)
	return l.GetMenuBaseInfoList(in)
}

// 添加系统基础菜单列表
func (s *MenuServer) AddMenuBase(ctx context.Context, in *pb.AddMenuBaseRequest) (*pb.NoDataResponse, error) {
	l := menulogic.NewAddMenuBaseLogic(ctx, s.svcCtx)
	return l.AddMenuBase(in)
}

// 获取用户动态路由树  -- 用于角色管理的设置权限
func (s *MenuServer) GetBaseMenuTree(ctx context.Context, in *pb.NoDataResponse) (*pb.GetBaseMenuTreeResponse, error) {
	l := menulogic.NewGetBaseMenuTreeLogic(ctx, s.svcCtx)
	return l.GetBaseMenuTree(in)
}

// 获取指定角色menu  -- 用于角色管理的设置权限
func (s *MenuServer) GetMenuAuthority(ctx context.Context, in *pb.GetMenuAuthorityRequest) (*pb.GetMenuAuthorityResponse, error) {
	l := menulogic.NewGetMenuAuthorityLogic(ctx, s.svcCtx)
	return l.GetMenuAuthority(in)
}

// 根据id获取系统菜单
func (s *MenuServer) GetBaseMenuById(ctx context.Context, in *pb.GetBaseMenuByIdRequest) (*pb.GetBaseMenuByIdResponse, error) {
	l := menulogic.NewGetBaseMenuByIdLogic(ctx, s.svcCtx)
	return l.GetBaseMenuById(in)
}

// 更新系统菜单
func (s *MenuServer) UpdateBaseMenu(ctx context.Context, in *pb.UpdateBaseMenuRequest) (*pb.NoDataResponse, error) {
	l := menulogic.NewUpdateBaseMenuLogic(ctx, s.svcCtx)
	return l.UpdateBaseMenu(in)
}

// 删除系统菜单
func (s *MenuServer) DeleteBaseMenu(ctx context.Context, in *pb.DeleteBaseMenuRequest) (*pb.NoDataResponse, error) {
	l := menulogic.NewDeleteBaseMenuLogic(ctx, s.svcCtx)
	return l.DeleteBaseMenu(in)
}
