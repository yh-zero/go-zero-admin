// Code generated by goctl. DO NOT EDIT.
// Source: applet.proto

package authority

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

	Authority interface {
		// 获取角色列表
		GetAuthorityList(ctx context.Context, in *GetAuthorityListRequest, opts ...grpc.CallOption) (*GetAuthorityListResponse, error)
		// 增加base_menu和角色关联关系 -- 用于角色管理的设置权限
		AddAuthorityMenu(ctx context.Context, in *AddAuthorityMenuRequest, opts ...grpc.CallOption) (*AddAuthorityMenuResponse, error)
		// 更新角色 -- 设为首页
		UpdateAuthority(ctx context.Context, in *UpdateAuthorityRequest, opts ...grpc.CallOption) (*UpdateAuthorityResponse, error)
		// 创建角色
		CreateAuthority(ctx context.Context, in *CreateAuthorityRequest, opts ...grpc.CallOption) (*CreateAuthorityResponse, error)
	}

	defaultAuthority struct {
		cli zrpc.Client
	}
)

func NewAuthority(cli zrpc.Client) Authority {
	return &defaultAuthority{
		cli: cli,
	}
}

// 获取角色列表
func (m *defaultAuthority) GetAuthorityList(ctx context.Context, in *GetAuthorityListRequest, opts ...grpc.CallOption) (*GetAuthorityListResponse, error) {
	client := pb.NewAuthorityClient(m.cli.Conn())
	return client.GetAuthorityList(ctx, in, opts...)
}

// 增加base_menu和角色关联关系 -- 用于角色管理的设置权限
func (m *defaultAuthority) AddAuthorityMenu(ctx context.Context, in *AddAuthorityMenuRequest, opts ...grpc.CallOption) (*AddAuthorityMenuResponse, error) {
	client := pb.NewAuthorityClient(m.cli.Conn())
	return client.AddAuthorityMenu(ctx, in, opts...)
}

// 更新角色 -- 设为首页
func (m *defaultAuthority) UpdateAuthority(ctx context.Context, in *UpdateAuthorityRequest, opts ...grpc.CallOption) (*UpdateAuthorityResponse, error) {
	client := pb.NewAuthorityClient(m.cli.Conn())
	return client.UpdateAuthority(ctx, in, opts...)
}

// 创建角色
func (m *defaultAuthority) CreateAuthority(ctx context.Context, in *CreateAuthorityRequest, opts ...grpc.CallOption) (*CreateAuthorityResponse, error) {
	client := pb.NewAuthorityClient(m.cli.Conn())
	return client.CreateAuthority(ctx, in, opts...)
}
