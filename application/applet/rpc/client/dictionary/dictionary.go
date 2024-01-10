// Code generated by goctl. DO NOT EDIT.
// Source: applet.proto

package dictionary

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
	DeleteAuthorityRequest          = pb.DeleteAuthorityRequest
	DeleteBaseMenuRequest           = pb.DeleteBaseMenuRequest
	DeleteUserRequest               = pb.DeleteUserRequest
	DictionaryListResponse          = pb.DictionaryListResponse
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
	SysDictionary                   = pb.SysDictionary
	SysDictionaryInfo               = pb.SysDictionaryInfo
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

	Dictionary interface {
		// 获取SysDictionary列表
		GetSysDictionaryList(ctx context.Context, in *NoDataResponse, opts ...grpc.CallOption) (*DictionaryListResponse, error)
	}

	defaultDictionary struct {
		cli zrpc.Client
	}
)

func NewDictionary(cli zrpc.Client) Dictionary {
	return &defaultDictionary{
		cli: cli,
	}
}

// 获取SysDictionary列表
func (m *defaultDictionary) GetSysDictionaryList(ctx context.Context, in *NoDataResponse, opts ...grpc.CallOption) (*DictionaryListResponse, error) {
	client := pb.NewDictionaryClient(m.cli.Conn())
	return client.GetSysDictionaryList(ctx, in, opts...)
}