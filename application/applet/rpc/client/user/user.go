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
	AddAuthorityMenuRequest          = pb.AddAuthorityMenuRequest
	AddMenuBaseRequest               = pb.AddMenuBaseRequest
	CasbinInfo                       = pb.CasbinInfo
	CreateApiRequest                 = pb.CreateApiRequest
	CreateAuthorityRequest           = pb.CreateAuthorityRequest
	CreateAuthorityResponse          = pb.CreateAuthorityResponse
	CreateSysDictionaryRequest       = pb.CreateSysDictionaryRequest
	DeleteApiRequest                 = pb.DeleteApiRequest
	DeleteApisByIdsRequest           = pb.DeleteApisByIdsRequest
	DeleteAuthorityRequest           = pb.DeleteAuthorityRequest
	DeleteBaseMenuRequest            = pb.DeleteBaseMenuRequest
	DeleteUserRequest                = pb.DeleteUserRequest
	DictionaryListResponse           = pb.DictionaryListResponse
	GetAllApiListResponse            = pb.GetAllApiListResponse
	GetApiListRequest                = pb.GetApiListRequest
	GetApiListResponse               = pb.GetApiListResponse
	GetAuthorityListRequest          = pb.GetAuthorityListRequest
	GetAuthorityListResponse         = pb.GetAuthorityListResponse
	GetBaseMenuByIdRequest           = pb.GetBaseMenuByIdRequest
	GetBaseMenuByIdResponse          = pb.GetBaseMenuByIdResponse
	GetBaseMenuTreeResponse          = pb.GetBaseMenuTreeResponse
	GetMenuAuthorityRequest          = pb.GetMenuAuthorityRequest
	GetMenuAuthorityResponse         = pb.GetMenuAuthorityResponse
	GetMenuBaseInfoListResponse      = pb.GetMenuBaseInfoListResponse
	GetMenuTreeRequest               = pb.GetMenuTreeRequest
	GetMenuTreeResponse              = pb.GetMenuTreeResponse
	GetPathByAuthorityIdRequest      = pb.GetPathByAuthorityIdRequest
	GetPathByAuthorityIdResponse     = pb.GetPathByAuthorityIdResponse
	GetSysDictionaryDetailsRequest   = pb.GetSysDictionaryDetailsRequest
	GetSysDictionaryDetailsResponse  = pb.GetSysDictionaryDetailsResponse
	GetSysDictionaryInfoListRequest  = pb.GetSysDictionaryInfoListRequest
	GetSysDictionaryInfoListResponse = pb.GetSysDictionaryInfoListResponse
	GetUserInfoRequest               = pb.GetUserInfoRequest
	GetUserInfoResponse              = pb.GetUserInfoResponse
	GetUserListRequest               = pb.GetUserListRequest
	GetUserListResponse              = pb.GetUserListResponse
	GetUserTokeRequest               = pb.GetUserTokeRequest
	GetUserTokeResponse              = pb.GetUserTokeResponse
	Meta                             = pb.Meta
	NoDataResponse                   = pb.NoDataResponse
	PageRequest                      = pb.PageRequest
	RegisterRequest                  = pb.RegisterRequest
	RegisterResponse                 = pb.RegisterResponse
	ResetUserPasswordRequest         = pb.ResetUserPasswordRequest
	SysApi                           = pb.SysApi
	SysAuthority                     = pb.SysAuthority
	SysBaseMenu                      = pb.SysBaseMenu
	SysBaseMenuBtn                   = pb.SysBaseMenuBtn
	SysBaseMenuParameter             = pb.SysBaseMenuParameter
	SysDictionary                    = pb.SysDictionary
	SysDictionaryInfo                = pb.SysDictionaryInfo
	SysMenu                          = pb.SysMenu
	UpdateApiRequest                 = pb.UpdateApiRequest
	UpdateAuthorityRequest           = pb.UpdateAuthorityRequest
	UpdateAuthorityResponse          = pb.UpdateAuthorityResponse
	UpdateBaseMenuRequest            = pb.UpdateBaseMenuRequest
	UpdateCasbinDataByApiIdsRequest  = pb.UpdateCasbinDataByApiIdsRequest
	UpdateCasbinDataRequest          = pb.UpdateCasbinDataRequest
	UpdateSysDictionaryRequest       = pb.UpdateSysDictionaryRequest
	UpdateUserAuthoritiesRequest     = pb.UpdateUserAuthoritiesRequest
	UpdateUserInfoRequest            = pb.UpdateUserInfoRequest
	UserInfo                         = pb.UserInfo

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
		UpdateUserInfo(ctx context.Context, in *UpdateUserInfoRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
		// 修改用户和角色的关系信息 -- 和上  在修改用户信息的时候请求
		UpdateUserAuthorities(ctx context.Context, in *UpdateUserAuthoritiesRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
		// 重置用户密码 默认密码：goZero
		ResetUserPassword(ctx context.Context, in *ResetUserPasswordRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
		// 删除用户
		DeleteUser(ctx context.Context, in *DeleteUserRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
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
func (m *defaultUser) UpdateUserInfo(ctx context.Context, in *UpdateUserInfoRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewUserClient(m.cli.Conn())
	return client.UpdateUserInfo(ctx, in, opts...)
}

// 修改用户和角色的关系信息 -- 和上  在修改用户信息的时候请求
func (m *defaultUser) UpdateUserAuthorities(ctx context.Context, in *UpdateUserAuthoritiesRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewUserClient(m.cli.Conn())
	return client.UpdateUserAuthorities(ctx, in, opts...)
}

// 重置用户密码 默认密码：goZero
func (m *defaultUser) ResetUserPassword(ctx context.Context, in *ResetUserPasswordRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewUserClient(m.cli.Conn())
	return client.ResetUserPassword(ctx, in, opts...)
}

// 删除用户
func (m *defaultUser) DeleteUser(ctx context.Context, in *DeleteUserRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewUserClient(m.cli.Conn())
	return client.DeleteUser(ctx, in, opts...)
}
