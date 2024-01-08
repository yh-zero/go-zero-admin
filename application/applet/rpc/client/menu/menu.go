// Code generated by goctl. DO NOT EDIT.
// Source: applet.proto

package menu

import (
	"context"

	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type (
	AddAuthorityMenuRequest       = pb.AddAuthorityMenuRequest
	AddAuthorityMenuResponse      = pb.AddAuthorityMenuResponse
	AddMenuBaseRequest            = pb.AddMenuBaseRequest
	AddMenuBaseResponse           = pb.AddMenuBaseResponse
	CasbinInfo                    = pb.CasbinInfo
	CreateApiRequest              = pb.CreateApiRequest
	CreateApiResponse             = pb.CreateApiResponse
	CreateAuthorityRequest        = pb.CreateAuthorityRequest
	CreateAuthorityResponse       = pb.CreateAuthorityResponse
	DeleteApiRequest              = pb.DeleteApiRequest
	DeleteApiResponse             = pb.DeleteApiResponse
	DeleteUserRequest             = pb.DeleteUserRequest
	DeleteUserResponse            = pb.DeleteUserResponse
	GetAllApiListRequest          = pb.GetAllApiListRequest
	GetAllApiListResponse         = pb.GetAllApiListResponse
	GetApiListRequest             = pb.GetApiListRequest
	GetApiListResponse            = pb.GetApiListResponse
	GetAuthorityListRequest       = pb.GetAuthorityListRequest
	GetAuthorityListResponse      = pb.GetAuthorityListResponse
	GetBaseMenuByIdRequest        = pb.GetBaseMenuByIdRequest
	GetBaseMenuByIdResponse       = pb.GetBaseMenuByIdResponse
	GetBaseMenuTreeRequest        = pb.GetBaseMenuTreeRequest
	GetBaseMenuTreeResponse       = pb.GetBaseMenuTreeResponse
	GetMenuAuthorityRequest       = pb.GetMenuAuthorityRequest
	GetMenuAuthorityResponse      = pb.GetMenuAuthorityResponse
	GetMenuBaseInfoListRequest    = pb.GetMenuBaseInfoListRequest
	GetMenuBaseInfoListResponse   = pb.GetMenuBaseInfoListResponse
	GetMenuTreeRequest            = pb.GetMenuTreeRequest
	GetMenuTreeResponse           = pb.GetMenuTreeResponse
	GetPathByAuthorityIdRequest   = pb.GetPathByAuthorityIdRequest
	GetPathByAuthorityIdResponse  = pb.GetPathByAuthorityIdResponse
	GetUserInfoRequest            = pb.GetUserInfoRequest
	GetUserInfoResponse           = pb.GetUserInfoResponse
	GetUserListRequest            = pb.GetUserListRequest
	GetUserListResponse           = pb.GetUserListResponse
	GetUserTokeRequest            = pb.GetUserTokeRequest
	GetUserTokeResponse           = pb.GetUserTokeResponse
	Meta                          = pb.Meta
	PageRequest                   = pb.PageRequest
	RegisterRequest               = pb.RegisterRequest
	RegisterResponse              = pb.RegisterResponse
	ResetUserPasswordRequest      = pb.ResetUserPasswordRequest
	ResetUserPasswordResponse     = pb.ResetUserPasswordResponse
	SysApi                        = pb.SysApi
	SysAuthority                  = pb.SysAuthority
	SysBaseMenu                   = pb.SysBaseMenu
	SysBaseMenuBtn                = pb.SysBaseMenuBtn
	SysBaseMenuParameter          = pb.SysBaseMenuParameter
	SysMenu                       = pb.SysMenu
	UpdateAuthorityRequest        = pb.UpdateAuthorityRequest
	UpdateAuthorityResponse       = pb.UpdateAuthorityResponse
	UpdateBaseMenuRequest         = pb.UpdateBaseMenuRequest
	UpdateBaseMenuResponse        = pb.UpdateBaseMenuResponse
	UpdateCasbinDataRequest       = pb.UpdateCasbinDataRequest
	UpdateCasbinDataResponse      = pb.UpdateCasbinDataResponse
	UpdateUserAuthoritiesRequest  = pb.UpdateUserAuthoritiesRequest
	UpdateUserAuthoritiesResponse = pb.UpdateUserAuthoritiesResponse
	UpdateUserInfoRequest         = pb.UpdateUserInfoRequest
	UpdateUserInfoResponse        = pb.UpdateUserInfoResponse
	UserInfo                      = pb.UserInfo

	Menu interface {
		// 获取菜单-路由
		GetMenuTree(ctx context.Context, in *GetMenuTreeRequest, opts ...grpc.CallOption) (*GetMenuTreeResponse, error)
		// 获取系统基础菜单列表
		GetMenuBaseInfoList(ctx context.Context, in *GetMenuBaseInfoListRequest, opts ...grpc.CallOption) (*GetMenuBaseInfoListResponse, error)
		// 添加系统基础菜单列表
		AddMenuBase(ctx context.Context, in *AddMenuBaseRequest, opts ...grpc.CallOption) (*AddMenuBaseResponse, error)
		// 获取用户动态路由树  -- 用于角色管理的设置权限
		GetBaseMenuTree(ctx context.Context, in *GetBaseMenuTreeRequest, opts ...grpc.CallOption) (*GetBaseMenuTreeResponse, error)
		// 获取指定角色menu  -- 用于角色管理的设置权限
		GetMenuAuthority(ctx context.Context, in *GetMenuAuthorityRequest, opts ...grpc.CallOption) (*GetMenuAuthorityResponse, error)
		// 根据id获取菜单
		GetBaseMenuById(ctx context.Context, in *GetBaseMenuByIdRequest, opts ...grpc.CallOption) (*GetBaseMenuByIdResponse, error)
		// 根据id获取菜单
		UpdateBaseMenu(ctx context.Context, in *UpdateBaseMenuRequest, opts ...grpc.CallOption) (*UpdateBaseMenuResponse, error)
	}

	defaultMenu struct {
		cli zrpc.Client
	}
)

func NewMenu(cli zrpc.Client) Menu {
	return &defaultMenu{
		cli: cli,
	}
}

// 获取菜单-路由
func (m *defaultMenu) GetMenuTree(ctx context.Context, in *GetMenuTreeRequest, opts ...grpc.CallOption) (*GetMenuTreeResponse, error) {
	client := pb.NewMenuClient(m.cli.Conn())
	return client.GetMenuTree(ctx, in, opts...)
}

// 获取系统基础菜单列表
func (m *defaultMenu) GetMenuBaseInfoList(ctx context.Context, in *GetMenuBaseInfoListRequest, opts ...grpc.CallOption) (*GetMenuBaseInfoListResponse, error) {
	client := pb.NewMenuClient(m.cli.Conn())
	return client.GetMenuBaseInfoList(ctx, in, opts...)
}

// 添加系统基础菜单列表
func (m *defaultMenu) AddMenuBase(ctx context.Context, in *AddMenuBaseRequest, opts ...grpc.CallOption) (*AddMenuBaseResponse, error) {
	client := pb.NewMenuClient(m.cli.Conn())
	return client.AddMenuBase(ctx, in, opts...)
}

// 获取用户动态路由树  -- 用于角色管理的设置权限
func (m *defaultMenu) GetBaseMenuTree(ctx context.Context, in *GetBaseMenuTreeRequest, opts ...grpc.CallOption) (*GetBaseMenuTreeResponse, error) {
	client := pb.NewMenuClient(m.cli.Conn())
	return client.GetBaseMenuTree(ctx, in, opts...)
}

// 获取指定角色menu  -- 用于角色管理的设置权限
func (m *defaultMenu) GetMenuAuthority(ctx context.Context, in *GetMenuAuthorityRequest, opts ...grpc.CallOption) (*GetMenuAuthorityResponse, error) {
	client := pb.NewMenuClient(m.cli.Conn())
	return client.GetMenuAuthority(ctx, in, opts...)
}

// 根据id获取菜单
func (m *defaultMenu) GetBaseMenuById(ctx context.Context, in *GetBaseMenuByIdRequest, opts ...grpc.CallOption) (*GetBaseMenuByIdResponse, error) {
	client := pb.NewMenuClient(m.cli.Conn())
	return client.GetBaseMenuById(ctx, in, opts...)
}

// 根据id获取菜单
func (m *defaultMenu) UpdateBaseMenu(ctx context.Context, in *UpdateBaseMenuRequest, opts ...grpc.CallOption) (*UpdateBaseMenuResponse, error) {
	client := pb.NewMenuClient(m.cli.Conn())
	return client.UpdateBaseMenu(ctx, in, opts...)
}
