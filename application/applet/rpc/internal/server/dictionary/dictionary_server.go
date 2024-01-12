// Code generated by goctl. DO NOT EDIT.
// Source: applet.proto

package server

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/logic/dictionary"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
)

type DictionaryServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedDictionaryServer
}

func NewDictionaryServer(svcCtx *svc.ServiceContext) *DictionaryServer {
	return &DictionaryServer{
		svcCtx: svcCtx,
	}
}

// 获取SysDictionary列表 -- all
func (s *DictionaryServer) GetSysDictionaryList(ctx context.Context, in *pb.NoDataResponse) (*pb.DictionaryListResponse, error) {
	l := dictionarylogic.NewGetSysDictionaryListLogic(ctx, s.svcCtx)
	return l.GetSysDictionaryList(in)
}

// 新建SysDictionary
func (s *DictionaryServer) CreateSysDictionary(ctx context.Context, in *pb.CreateSysDictionaryRequest) (*pb.NoDataResponse, error) {
	l := dictionarylogic.NewCreateSysDictionaryLogic(ctx, s.svcCtx)
	return l.CreateSysDictionary(in)
}

// 根据ID或者type获取SysDictionary
func (s *DictionaryServer) GetSysDictionaryDetails(ctx context.Context, in *pb.GetSysDictionaryDetailsRequest) (*pb.GetSysDictionaryDetailsResponse, error) {
	l := dictionarylogic.NewGetSysDictionaryDetailsLogic(ctx, s.svcCtx)
	return l.GetSysDictionaryDetails(in)
}

// 更新SysDictionary
func (s *DictionaryServer) UpdateSysDictionary(ctx context.Context, in *pb.UpdateSysDictionaryRequest) (*pb.NoDataResponse, error) {
	l := dictionarylogic.NewUpdateSysDictionaryLogic(ctx, s.svcCtx)
	return l.UpdateSysDictionary(in)
}

// 更新SysDictionary
func (s *DictionaryServer) DeleteSysDictionary(ctx context.Context, in *pb.DeleteSysDictionaryRequest) (*pb.NoDataResponse, error) {
	l := dictionarylogic.NewDeleteSysDictionaryLogic(ctx, s.svcCtx)
	return l.DeleteSysDictionary(in)
}

// 获取SysDictionaryInfo列表 -- 分页带搜索
func (s *DictionaryServer) GetSysDictionaryInfoList(ctx context.Context, in *pb.GetSysDictionaryInfoListRequest) (*pb.GetSysDictionaryInfoListResponse, error) {
	l := dictionarylogic.NewGetSysDictionaryInfoListLogic(ctx, s.svcCtx)
	return l.GetSysDictionaryInfoList(in)
}

// 根据id获取SysDictionaryInfo详情
func (s *DictionaryServer) GetSysDictionaryInfoListDetailsById(ctx context.Context, in *pb.GetSysDictionaryInfoListDetailsByIdRequest) (*pb.GetSysDictionaryInfoListDetailsByIdResponse, error) {
	l := dictionarylogic.NewGetSysDictionaryInfoListDetailsByIdLogic(ctx, s.svcCtx)
	return l.GetSysDictionaryInfoListDetailsById(in)
}

// 更新SysDictionaryInfo
func (s *DictionaryServer) UpdateSysDictionaryInfo(ctx context.Context, in *pb.UpdateSysDictionaryInfoRequest) (*pb.NoDataResponse, error) {
	l := dictionarylogic.NewUpdateSysDictionaryInfoLogic(ctx, s.svcCtx)
	return l.UpdateSysDictionaryInfo(in)
}

// 创建SysDictionaryInfo
func (s *DictionaryServer) CreateSysDictionaryInfo(ctx context.Context, in *pb.CreateSysDictionaryInfoRequest) (*pb.NoDataResponse, error) {
	l := dictionarylogic.NewCreateSysDictionaryInfoLogic(ctx, s.svcCtx)
	return l.CreateSysDictionaryInfo(in)
}

// 删除SysDictionaryInfo
func (s *DictionaryServer) DeleteSysDictionaryInfo(ctx context.Context, in *pb.DeleteSysDictionaryInfoRequest) (*pb.NoDataResponse, error) {
	l := dictionarylogic.NewDeleteSysDictionaryInfoLogic(ctx, s.svcCtx)
	return l.DeleteSysDictionaryInfo(in)
}
