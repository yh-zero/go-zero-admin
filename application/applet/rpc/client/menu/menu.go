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
	AddAuthorityMenuRequest                     = pb.AddAuthorityMenuRequest
	AddMenuBaseRequest                          = pb.AddMenuBaseRequest
	CasbinInfo                                  = pb.CasbinInfo
	CreateApiRequest                            = pb.CreateApiRequest
	CreateAuthorityRequest                      = pb.CreateAuthorityRequest
	CreateAuthorityResponse                     = pb.CreateAuthorityResponse
	CreateSysDictionaryInfoRequest              = pb.CreateSysDictionaryInfoRequest
	CreateSysDictionaryRequest                  = pb.CreateSysDictionaryRequest
	DeleteApiRequest                            = pb.DeleteApiRequest
	DeleteApisByIdsRequest                      = pb.DeleteApisByIdsRequest
	DeleteAuthorityRequest                      = pb.DeleteAuthorityRequest
	DeleteBaseMenuRequest                       = pb.DeleteBaseMenuRequest
	DeleteSysDictionaryInfoRequest              = pb.DeleteSysDictionaryInfoRequest
	DeleteSysDictionaryRequest                  = pb.DeleteSysDictionaryRequest
	DeleteUserRequest                           = pb.DeleteUserRequest
	DictionaryListResponse                      = pb.DictionaryListResponse
	GetAllApiListResponse                       = pb.GetAllApiListResponse
	GetApiListRequest                           = pb.GetApiListRequest
	GetApiListResponse                          = pb.GetApiListResponse
	GetAuthorityListRequest                     = pb.GetAuthorityListRequest
	GetAuthorityListResponse                    = pb.GetAuthorityListResponse
	GetBaseMenuByIdRequest                      = pb.GetBaseMenuByIdRequest
	GetBaseMenuByIdResponse                     = pb.GetBaseMenuByIdResponse
	GetBaseMenuTreeResponse                     = pb.GetBaseMenuTreeResponse
	GetMenuAuthorityRequest                     = pb.GetMenuAuthorityRequest
	GetMenuAuthorityResponse                    = pb.GetMenuAuthorityResponse
	GetMenuBaseInfoListResponse                 = pb.GetMenuBaseInfoListResponse
	GetMenuTreeRequest                          = pb.GetMenuTreeRequest
	GetMenuTreeResponse                         = pb.GetMenuTreeResponse
	GetPathByAuthorityIdRequest                 = pb.GetPathByAuthorityIdRequest
	GetPathByAuthorityIdResponse                = pb.GetPathByAuthorityIdResponse
	GetSysDictionaryDetailsRequest              = pb.GetSysDictionaryDetailsRequest
	GetSysDictionaryDetailsResponse             = pb.GetSysDictionaryDetailsResponse
	GetSysDictionaryInfoListDetailsByIdRequest  = pb.GetSysDictionaryInfoListDetailsByIdRequest
	GetSysDictionaryInfoListDetailsByIdResponse = pb.GetSysDictionaryInfoListDetailsByIdResponse
	GetSysDictionaryInfoListRequest             = pb.GetSysDictionaryInfoListRequest
	GetSysDictionaryInfoListResponse            = pb.GetSysDictionaryInfoListResponse
	GetUserInfoRequest                          = pb.GetUserInfoRequest
	GetUserInfoResponse                         = pb.GetUserInfoResponse
	GetUserListRequest                          = pb.GetUserListRequest
	GetUserListResponse                         = pb.GetUserListResponse
	GetUserTokeRequest                          = pb.GetUserTokeRequest
	GetUserTokeResponse                         = pb.GetUserTokeResponse
	Meta                                        = pb.Meta
	NoDataResponse                              = pb.NoDataResponse
	PageRequest                                 = pb.PageRequest
	RegisterRequest                             = pb.RegisterRequest
	RegisterResponse                            = pb.RegisterResponse
	ResetUserPasswordRequest                    = pb.ResetUserPasswordRequest
	SysApi                                      = pb.SysApi
	SysAuthority                                = pb.SysAuthority
	SysBaseMenu                                 = pb.SysBaseMenu
	SysBaseMenuBtn                              = pb.SysBaseMenuBtn
	SysBaseMenuParameter                        = pb.SysBaseMenuParameter
	SysDictionary                               = pb.SysDictionary
	SysDictionaryInfo                           = pb.SysDictionaryInfo
	SysMenu                                     = pb.SysMenu
	UpdateApiRequest                            = pb.UpdateApiRequest
	UpdateAuthorityRequest                      = pb.UpdateAuthorityRequest
	UpdateAuthorityResponse                     = pb.UpdateAuthorityResponse
	UpdateBaseMenuRequest                       = pb.UpdateBaseMenuRequest
	UpdateCasbinDataByApiIdsRequest             = pb.UpdateCasbinDataByApiIdsRequest
	UpdateCasbinDataRequest                     = pb.UpdateCasbinDataRequest
	UpdateSysDictionaryInfoRequest              = pb.UpdateSysDictionaryInfoRequest
	UpdateSysDictionaryRequest                  = pb.UpdateSysDictionaryRequest
	UpdateUserAuthoritiesRequest                = pb.UpdateUserAuthoritiesRequest
	UpdateUserInfoRequest                       = pb.UpdateUserInfoRequest
	UserInfo                                    = pb.UserInfo

	Menu interface {
		// 获取菜单-路由
		GetMenuTree(ctx context.Context, in *GetMenuTreeRequest, opts ...grpc.CallOption) (*GetMenuTreeResponse, error)
		// 获取系统基础菜单列表
		GetMenuBaseInfoList(ctx context.Context, in *NoDataResponse, opts ...grpc.CallOption) (*GetMenuBaseInfoListResponse, error)
		// 添加系统基础菜单列表
		AddMenuBase(ctx context.Context, in *AddMenuBaseRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
		// 获取用户动态路由树  -- 用于角色管理的设置权限
		GetBaseMenuTree(ctx context.Context, in *NoDataResponse, opts ...grpc.CallOption) (*GetBaseMenuTreeResponse, error)
		// 获取指定角色menu  -- 用于角色管理的设置权限
		GetMenuAuthority(ctx context.Context, in *GetMenuAuthorityRequest, opts ...grpc.CallOption) (*GetMenuAuthorityResponse, error)
		// 根据id获取系统菜单
		GetBaseMenuById(ctx context.Context, in *GetBaseMenuByIdRequest, opts ...grpc.CallOption) (*GetBaseMenuByIdResponse, error)
		// 更新系统菜单
		UpdateBaseMenu(ctx context.Context, in *UpdateBaseMenuRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
		// 删除系统菜单
		DeleteBaseMenu(ctx context.Context, in *DeleteBaseMenuRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
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
func (m *defaultMenu) GetMenuBaseInfoList(ctx context.Context, in *NoDataResponse, opts ...grpc.CallOption) (*GetMenuBaseInfoListResponse, error) {
	client := pb.NewMenuClient(m.cli.Conn())
	return client.GetMenuBaseInfoList(ctx, in, opts...)
}

// 添加系统基础菜单列表
func (m *defaultMenu) AddMenuBase(ctx context.Context, in *AddMenuBaseRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewMenuClient(m.cli.Conn())
	return client.AddMenuBase(ctx, in, opts...)
}

// 获取用户动态路由树  -- 用于角色管理的设置权限
func (m *defaultMenu) GetBaseMenuTree(ctx context.Context, in *NoDataResponse, opts ...grpc.CallOption) (*GetBaseMenuTreeResponse, error) {
	client := pb.NewMenuClient(m.cli.Conn())
	return client.GetBaseMenuTree(ctx, in, opts...)
}

// 获取指定角色menu  -- 用于角色管理的设置权限
func (m *defaultMenu) GetMenuAuthority(ctx context.Context, in *GetMenuAuthorityRequest, opts ...grpc.CallOption) (*GetMenuAuthorityResponse, error) {
	client := pb.NewMenuClient(m.cli.Conn())
	return client.GetMenuAuthority(ctx, in, opts...)
}

// 根据id获取系统菜单
func (m *defaultMenu) GetBaseMenuById(ctx context.Context, in *GetBaseMenuByIdRequest, opts ...grpc.CallOption) (*GetBaseMenuByIdResponse, error) {
	client := pb.NewMenuClient(m.cli.Conn())
	return client.GetBaseMenuById(ctx, in, opts...)
}

// 更新系统菜单
func (m *defaultMenu) UpdateBaseMenu(ctx context.Context, in *UpdateBaseMenuRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewMenuClient(m.cli.Conn())
	return client.UpdateBaseMenu(ctx, in, opts...)
}

// 删除系统菜单
func (m *defaultMenu) DeleteBaseMenu(ctx context.Context, in *DeleteBaseMenuRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewMenuClient(m.cli.Conn())
	return client.DeleteBaseMenu(ctx, in, opts...)
}
