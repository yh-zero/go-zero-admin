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
	AddAuthorityMenuRequest                     = pb.AddAuthorityMenuRequest
	AddMenuBaseRequest                          = pb.AddMenuBaseRequest
	CasbinInfo                                  = pb.CasbinInfo
	CreateApiRequest                            = pb.CreateApiRequest
	CreateAuthorityRequest                      = pb.CreateAuthorityRequest
	CreateAuthorityResponse                     = pb.CreateAuthorityResponse
	CreateSysDictionaryRequest                  = pb.CreateSysDictionaryRequest
	DeleteApiRequest                            = pb.DeleteApiRequest
	DeleteApisByIdsRequest                      = pb.DeleteApisByIdsRequest
	DeleteAuthorityRequest                      = pb.DeleteAuthorityRequest
	DeleteBaseMenuRequest                       = pb.DeleteBaseMenuRequest
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
	UpdateSysDictionaryRequest                  = pb.UpdateSysDictionaryRequest
	UpdateUserAuthoritiesRequest                = pb.UpdateUserAuthoritiesRequest
	UpdateUserInfoRequest                       = pb.UpdateUserInfoRequest
	UserInfo                                    = pb.UserInfo

	Api interface {
		// 获取API列表
		GetApiList(ctx context.Context, in *GetApiListRequest, opts ...grpc.CallOption) (*GetApiListResponse, error)
		// 创建/添加 API列表
		CreateApi(ctx context.Context, in *CreateApiRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
		// 删除API列表
		DeleteApi(ctx context.Context, in *DeleteApiRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
		// 获取全部API列表
		GetAllApiList(ctx context.Context, in *NoDataResponse, opts ...grpc.CallOption) (*GetAllApiListResponse, error)
		// 删除多条api
		DeleteApisByIds(ctx context.Context, in *DeleteApisByIdsRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
		// 更新api
		UpdateApi(ctx context.Context, in *UpdateApiRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
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
func (m *defaultApi) CreateApi(ctx context.Context, in *CreateApiRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewApiClient(m.cli.Conn())
	return client.CreateApi(ctx, in, opts...)
}

// 删除API列表
func (m *defaultApi) DeleteApi(ctx context.Context, in *DeleteApiRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewApiClient(m.cli.Conn())
	return client.DeleteApi(ctx, in, opts...)
}

// 获取全部API列表
func (m *defaultApi) GetAllApiList(ctx context.Context, in *NoDataResponse, opts ...grpc.CallOption) (*GetAllApiListResponse, error) {
	client := pb.NewApiClient(m.cli.Conn())
	return client.GetAllApiList(ctx, in, opts...)
}

// 删除多条api
func (m *defaultApi) DeleteApisByIds(ctx context.Context, in *DeleteApisByIdsRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewApiClient(m.cli.Conn())
	return client.DeleteApisByIds(ctx, in, opts...)
}

// 更新api
func (m *defaultApi) UpdateApi(ctx context.Context, in *UpdateApiRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewApiClient(m.cli.Conn())
	return client.UpdateApi(ctx, in, opts...)
}
