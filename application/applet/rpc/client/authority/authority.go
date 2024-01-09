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
	AddAuthorityMenuRequest         = pb.AddAuthorityMenuRequest
	AddMenuBaseRequest              = pb.AddMenuBaseRequest
	CasbinInfo                      = pb.CasbinInfo
	CreateApiRequest                = pb.CreateApiRequest
	CreateAuthorityRequest          = pb.CreateAuthorityRequest
	CreateAuthorityResponse         = pb.CreateAuthorityResponse
	DeleteApiRequest                = pb.DeleteApiRequest
	DeleteApisByIdsRequest          = pb.DeleteApisByIdsRequest
	DeleteUserRequest               = pb.DeleteUserRequest
	GetAllApiListResponse           = pb.GetAllApiListResponse
	GetApiListRequest               = pb.GetApiListRequest
	GetApiListResponse              = pb.GetApiListResponse
	GetAuthorityListRequest         = pb.GetAuthorityListRequest
	GetAuthorityListResponse        = pb.GetAuthorityListResponse
	GetBaseMenuByIdRequest          = pb.GetBaseMenuByIdRequest
	GetBaseMenuByIdResponse         = pb.GetBaseMenuByIdResponse
	GetBaseMenuTreeResponse         = pb.GetBaseMenuTreeResponse
	GetMenuAuthorityRequest         = pb.GetMenuAuthorityRequest
	GetMenuAuthorityResponse        = pb.GetMenuAuthorityResponse
	GetMenuBaseInfoListResponse     = pb.GetMenuBaseInfoListResponse
	GetMenuTreeRequest              = pb.GetMenuTreeRequest
	GetMenuTreeResponse             = pb.GetMenuTreeResponse
	GetPathByAuthorityIdRequest     = pb.GetPathByAuthorityIdRequest
	GetPathByAuthorityIdResponse    = pb.GetPathByAuthorityIdResponse
	GetUserInfoRequest              = pb.GetUserInfoRequest
	GetUserInfoResponse             = pb.GetUserInfoResponse
	GetUserListRequest              = pb.GetUserListRequest
	GetUserListResponse             = pb.GetUserListResponse
	GetUserTokeRequest              = pb.GetUserTokeRequest
	GetUserTokeResponse             = pb.GetUserTokeResponse
	Meta                            = pb.Meta
	NoDataResponse                  = pb.NoDataResponse
	PageRequest                     = pb.PageRequest
	RegisterRequest                 = pb.RegisterRequest
	RegisterResponse                = pb.RegisterResponse
	ResetUserPasswordRequest        = pb.ResetUserPasswordRequest
	SysApi                          = pb.SysApi
	SysAuthority                    = pb.SysAuthority
	SysBaseMenu                     = pb.SysBaseMenu
	SysBaseMenuBtn                  = pb.SysBaseMenuBtn
	SysBaseMenuParameter            = pb.SysBaseMenuParameter
	SysMenu                         = pb.SysMenu
	UpdateApiRequest                = pb.UpdateApiRequest
	UpdateAuthorityRequest          = pb.UpdateAuthorityRequest
	UpdateAuthorityResponse         = pb.UpdateAuthorityResponse
	UpdateBaseMenuRequest           = pb.UpdateBaseMenuRequest
	UpdateCasbinDataByApiIdsRequest = pb.UpdateCasbinDataByApiIdsRequest
	UpdateCasbinDataRequest         = pb.UpdateCasbinDataRequest
	UpdateUserAuthoritiesRequest    = pb.UpdateUserAuthoritiesRequest
	UpdateUserInfoRequest           = pb.UpdateUserInfoRequest
	UserInfo                        = pb.UserInfo

	Authority interface {
		// 获取角色列表
		GetAuthorityList(ctx context.Context, in *GetAuthorityListRequest, opts ...grpc.CallOption) (*GetAuthorityListResponse, error)
		// 增加base_menu和角色关联关系 -- 用于角色管理的设置权限
		AddAuthorityMenu(ctx context.Context, in *AddAuthorityMenuRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
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
func (m *defaultAuthority) AddAuthorityMenu(ctx context.Context, in *AddAuthorityMenuRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
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
