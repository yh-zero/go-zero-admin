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
	UpdateCasbinDataByApiIdsResponse            = pb.UpdateCasbinDataByApiIdsResponse
	UpdateCasbinDataRequest                     = pb.UpdateCasbinDataRequest
	UpdateSysDictionaryInfoRequest              = pb.UpdateSysDictionaryInfoRequest
	UpdateSysDictionaryRequest                  = pb.UpdateSysDictionaryRequest
	UpdateUserAuthoritiesRequest                = pb.UpdateUserAuthoritiesRequest
	UpdateUserInfoRequest                       = pb.UpdateUserInfoRequest
	UserInfo                                    = pb.UserInfo

	Dictionary interface {
		// 获取SysDictionary列表 -- all
		GetSysDictionaryList(ctx context.Context, in *NoDataResponse, opts ...grpc.CallOption) (*DictionaryListResponse, error)
		// 新建SysDictionary
		CreateSysDictionary(ctx context.Context, in *CreateSysDictionaryRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
		// 根据ID或者type获取SysDictionary
		GetSysDictionaryDetails(ctx context.Context, in *GetSysDictionaryDetailsRequest, opts ...grpc.CallOption) (*GetSysDictionaryDetailsResponse, error)
		// 更新SysDictionary
		UpdateSysDictionary(ctx context.Context, in *UpdateSysDictionaryRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
		// 更新SysDictionary
		DeleteSysDictionary(ctx context.Context, in *DeleteSysDictionaryRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
		// 获取SysDictionaryInfo列表 -- 分页带搜索
		GetSysDictionaryInfoList(ctx context.Context, in *GetSysDictionaryInfoListRequest, opts ...grpc.CallOption) (*GetSysDictionaryInfoListResponse, error)
		// 根据id获取SysDictionaryInfo详情
		GetSysDictionaryInfoListDetailsById(ctx context.Context, in *GetSysDictionaryInfoListDetailsByIdRequest, opts ...grpc.CallOption) (*GetSysDictionaryInfoListDetailsByIdResponse, error)
		// 更新SysDictionaryInfo
		UpdateSysDictionaryInfo(ctx context.Context, in *UpdateSysDictionaryInfoRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
		// 创建SysDictionaryInfo
		CreateSysDictionaryInfo(ctx context.Context, in *CreateSysDictionaryInfoRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
		// 删除SysDictionaryInfo
		DeleteSysDictionaryInfo(ctx context.Context, in *DeleteSysDictionaryInfoRequest, opts ...grpc.CallOption) (*NoDataResponse, error)
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

// 获取SysDictionary列表 -- all
func (m *defaultDictionary) GetSysDictionaryList(ctx context.Context, in *NoDataResponse, opts ...grpc.CallOption) (*DictionaryListResponse, error) {
	client := pb.NewDictionaryClient(m.cli.Conn())
	return client.GetSysDictionaryList(ctx, in, opts...)
}

// 新建SysDictionary
func (m *defaultDictionary) CreateSysDictionary(ctx context.Context, in *CreateSysDictionaryRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewDictionaryClient(m.cli.Conn())
	return client.CreateSysDictionary(ctx, in, opts...)
}

// 根据ID或者type获取SysDictionary
func (m *defaultDictionary) GetSysDictionaryDetails(ctx context.Context, in *GetSysDictionaryDetailsRequest, opts ...grpc.CallOption) (*GetSysDictionaryDetailsResponse, error) {
	client := pb.NewDictionaryClient(m.cli.Conn())
	return client.GetSysDictionaryDetails(ctx, in, opts...)
}

// 更新SysDictionary
func (m *defaultDictionary) UpdateSysDictionary(ctx context.Context, in *UpdateSysDictionaryRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewDictionaryClient(m.cli.Conn())
	return client.UpdateSysDictionary(ctx, in, opts...)
}

// 更新SysDictionary
func (m *defaultDictionary) DeleteSysDictionary(ctx context.Context, in *DeleteSysDictionaryRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewDictionaryClient(m.cli.Conn())
	return client.DeleteSysDictionary(ctx, in, opts...)
}

// 获取SysDictionaryInfo列表 -- 分页带搜索
func (m *defaultDictionary) GetSysDictionaryInfoList(ctx context.Context, in *GetSysDictionaryInfoListRequest, opts ...grpc.CallOption) (*GetSysDictionaryInfoListResponse, error) {
	client := pb.NewDictionaryClient(m.cli.Conn())
	return client.GetSysDictionaryInfoList(ctx, in, opts...)
}

// 根据id获取SysDictionaryInfo详情
func (m *defaultDictionary) GetSysDictionaryInfoListDetailsById(ctx context.Context, in *GetSysDictionaryInfoListDetailsByIdRequest, opts ...grpc.CallOption) (*GetSysDictionaryInfoListDetailsByIdResponse, error) {
	client := pb.NewDictionaryClient(m.cli.Conn())
	return client.GetSysDictionaryInfoListDetailsById(ctx, in, opts...)
}

// 更新SysDictionaryInfo
func (m *defaultDictionary) UpdateSysDictionaryInfo(ctx context.Context, in *UpdateSysDictionaryInfoRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewDictionaryClient(m.cli.Conn())
	return client.UpdateSysDictionaryInfo(ctx, in, opts...)
}

// 创建SysDictionaryInfo
func (m *defaultDictionary) CreateSysDictionaryInfo(ctx context.Context, in *CreateSysDictionaryInfoRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewDictionaryClient(m.cli.Conn())
	return client.CreateSysDictionaryInfo(ctx, in, opts...)
}

// 删除SysDictionaryInfo
func (m *defaultDictionary) DeleteSysDictionaryInfo(ctx context.Context, in *DeleteSysDictionaryInfoRequest, opts ...grpc.CallOption) (*NoDataResponse, error) {
	client := pb.NewDictionaryClient(m.cli.Conn())
	return client.DeleteSysDictionaryInfo(ctx, in, opts...)
}
