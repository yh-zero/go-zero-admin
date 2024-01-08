// Code generated by goctl. DO NOT EDIT.
// Source: applet.proto

package user

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

	User interface {
		// 获取用户信息
		GetUserInfo(ctx context.Context, in *GetUserInfoRequest, opts ...grpc.CallOption) (*GetUserInfoResponse, error)
		// 获取Token
		GetUserToke(ctx context.Context, in *GetUserTokeRequest, opts ...grpc.CallOption) (*GetUserTokeResponse, error)
		// 分页获取用户列表
		GetUserList(ctx context.Context, in *GetUserListRequest, opts ...grpc.CallOption) (*GetUserListResponse, error)
		// 新增（注册）用户 - 管理员
		Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error)
		// 修改用户信息
		UpdateUserInfo(ctx context.Context, in *UpdateUserInfoRequest, opts ...grpc.CallOption) (*UpdateUserInfoResponse, error)
		// 修改用户和角色的关系信息 -- 和上  在修改用户信息的时候请求
		UpdateUserAuthorities(ctx context.Context, in *UpdateUserAuthoritiesRequest, opts ...grpc.CallOption) (*UpdateUserAuthoritiesResponse, error)
		// 重置用户密码 默认密码：goZero
		ResetUserPassword(ctx context.Context, in *ResetUserPasswordRequest, opts ...grpc.CallOption) (*ResetUserPasswordResponse, error)
	}

	defaultUser struct {
		cli zrpc.Client
	}
)

func NewUser(cli zrpc.Client) User {
	return &defaultUser{
		cli: cli,
	}
}

// 获取用户信息
func (m *defaultUser) GetUserInfo(ctx context.Context, in *GetUserInfoRequest, opts ...grpc.CallOption) (*GetUserInfoResponse, error) {
	client := pb.NewUserClient(m.cli.Conn())
	return client.GetUserInfo(ctx, in, opts...)
}

// 获取Token
func (m *defaultUser) GetUserToke(ctx context.Context, in *GetUserTokeRequest, opts ...grpc.CallOption) (*GetUserTokeResponse, error) {
	client := pb.NewUserClient(m.cli.Conn())
	return client.GetUserToke(ctx, in, opts...)
}

// 分页获取用户列表
func (m *defaultUser) GetUserList(ctx context.Context, in *GetUserListRequest, opts ...grpc.CallOption) (*GetUserListResponse, error) {
	client := pb.NewUserClient(m.cli.Conn())
	return client.GetUserList(ctx, in, opts...)
}

// 新增（注册）用户 - 管理员
func (m *defaultUser) Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error) {
	client := pb.NewUserClient(m.cli.Conn())
	return client.Register(ctx, in, opts...)
}

// 修改用户信息
func (m *defaultUser) UpdateUserInfo(ctx context.Context, in *UpdateUserInfoRequest, opts ...grpc.CallOption) (*UpdateUserInfoResponse, error) {
	client := pb.NewUserClient(m.cli.Conn())
	return client.UpdateUserInfo(ctx, in, opts...)
}

// 修改用户和角色的关系信息 -- 和上  在修改用户信息的时候请求
func (m *defaultUser) UpdateUserAuthorities(ctx context.Context, in *UpdateUserAuthoritiesRequest, opts ...grpc.CallOption) (*UpdateUserAuthoritiesResponse, error) {
	client := pb.NewUserClient(m.cli.Conn())
	return client.UpdateUserAuthorities(ctx, in, opts...)
}

// 重置用户密码 默认密码：goZero
func (m *defaultUser) ResetUserPassword(ctx context.Context, in *ResetUserPasswordRequest, opts ...grpc.CallOption) (*ResetUserPasswordResponse, error) {
	client := pb.NewUserClient(m.cli.Conn())
	return client.ResetUserPassword(ctx, in, opts...)
}
