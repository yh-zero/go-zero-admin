// Code generated by goctl. DO NOT EDIT.
// Source: applet.proto

package casbin

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
	UpdateSysDictionaryInfoRequest              = pb.UpdateSysDictionaryInfoRequest
	UpdateSysDictionaryRequest                  = pb.UpdateSysDictionaryRequest
	UpdateUserAuthoritiesRequest                = pb.UpdateUserAuthoritiesRequest
	UpdateUserInfoRequest                       = pb.UpdateUserInfoRequest
	UserInfo                                    = pb.UserInfo

	Casbin interface {
		// 根据角色id获取对应的casbin数据
		GetPathByAuthorityId(ctx context.Context, in *GetPathByAuthorityIdRequest, opts ...grpc.CallOption) (*GetPathByAuthorityIdResponse, error)
		// 更新一个角色的对应的casbin数据
		UpdateCasbinData(ctx context.Context, in *UpdateCasbinDataRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
		// 更新一个角色的对应的casbin数据 用api的ids 查数据
		UpdateCasbinDataByApiIds(ctx context.Context, in *UpdateCasbinDataByApiIdsRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
	}

	defaultCasbin struct {
		cli zrpc.Client
	}
)

func NewCasbin(cli zrpc.Client) Casbin {
	return &defaultCasbin{
		cli: cli,
	}
}

// 根据角色id获取对应的casbin数据
func (m *defaultCasbin) GetPathByAuthorityId(ctx context.Context, in *GetPathByAuthorityIdRequest, opts ...grpc.CallOption) (*GetPathByAuthorityIdResponse, error) {
	client := pb.NewCasbinClient(m.cli.Conn())
	return client.GetPathByAuthorityId(ctx, in, opts...)
}

// 更新一个角色的对应的casbin数据
func (m *defaultCasbin) UpdateCasbinData(ctx context.Context, in *UpdateCasbinDataRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewCasbinClient(m.cli.Conn())
	return client.UpdateCasbinData(ctx, in, opts...)
}

// 更新一个角色的对应的casbin数据 用api的ids 查数据
func (m *defaultCasbin) UpdateCasbinDataByApiIds(ctx context.Context, in *UpdateCasbinDataByApiIdsRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewCasbinClient(m.cli.Conn())
	return client.UpdateCasbinDataByApiIds(ctx, in, opts...)
}
