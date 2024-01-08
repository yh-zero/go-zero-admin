// Code generated by goctl. DO NOT EDIT.
// Source: applet.proto

package api

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
	DeleteApisByIdsRequest        = pb.DeleteApisByIdsRequest
	DeleteApisByIdsResponse       = pb.DeleteApisByIdsResponse
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

	Api interface {
		// 获取API列表
		GetApiList(ctx context.Context, in *GetApiListRequest, opts ...grpc.CallOption) (*GetApiListResponse, error)
		// 创建/添加 API列表
		CreateApi(ctx context.Context, in *CreateApiRequest, opts ...grpc.CallOption) (*CreateApiResponse, error)
		// 删除API列表
		DeleteApi(ctx context.Context, in *DeleteApiRequest, opts ...grpc.CallOption) (*DeleteApiResponse, error)
		// 获取全部API列表
		GetAllApiList(ctx context.Context, in *GetAllApiListRequest, opts ...grpc.CallOption) (*GetAllApiListResponse, error)
		// 删除多条api
		DeleteApisByIds(ctx context.Context, in *DeleteApisByIdsRequest, opts ...grpc.CallOption) (*DeleteApisByIdsResponse, error)
	}

	defaultApi struct {
		cli zrpc.Client
	}
)

func NewApi(cli zrpc.Client) Api {
	return &defaultApi{
		cli: cli,
	}
}

// 获取API列表
func (m *defaultApi) GetApiList(ctx context.Context, in *GetApiListRequest, opts ...grpc.CallOption) (*GetApiListResponse, error) {
	client := pb.NewApiClient(m.cli.Conn())
	return client.GetApiList(ctx, in, opts...)
}

// 创建/添加 API列表
func (m *defaultApi) CreateApi(ctx context.Context, in *CreateApiRequest, opts ...grpc.CallOption) (*CreateApiResponse, error) {
	client := pb.NewApiClient(m.cli.Conn())
	return client.CreateApi(ctx, in, opts...)
}

// 删除API列表
func (m *defaultApi) DeleteApi(ctx context.Context, in *DeleteApiRequest, opts ...grpc.CallOption) (*DeleteApiResponse, error) {
	client := pb.NewApiClient(m.cli.Conn())
	return client.DeleteApi(ctx, in, opts...)
}

// 获取全部API列表
func (m *defaultApi) GetAllApiList(ctx context.Context, in *GetAllApiListRequest, opts ...grpc.CallOption) (*GetAllApiListResponse, error) {
	client := pb.NewApiClient(m.cli.Conn())
	return client.GetAllApiList(ctx, in, opts...)
}

// 删除多条api
func (m *defaultApi) DeleteApisByIds(ctx context.Context, in *DeleteApisByIdsRequest, opts ...grpc.CallOption) (*DeleteApisByIdsResponse, error) {
	client := pb.NewApiClient(m.cli.Conn())
	return client.DeleteApisByIds(ctx, in, opts...)
}
