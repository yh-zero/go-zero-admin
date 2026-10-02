// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package files

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFileListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 按数据范围分页查询文件
func NewGetFileListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileListLogic {
	return &GetFileListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetFileListLogic) GetFileList(req *types.FileListRequest) (resp *types.FileListResponse, err error) {
	page, size := req.PageNo, req.PageSize
	if page == 0 {
		page = l.svcCtx.Config.Page.PageNo
	}
	if size == 0 {
		size = l.svcCtx.Config.Page.PageSize
	}
	response, err := l.svcCtx.AppletFileRPC.GetFileList(l.ctx, &pb.FileListRequest{Actor: actorSession(l.ctx), PageRequest: &pb.PageRequest{PageNo: page, PageSize: size}, Name: req.Name, Status: req.Status})
	if err != nil {
		return nil, err
	}
	list := make([]types.FileResource, 0, len(response.List))
	for _, file := range response.List {
		list = append(list, fileType(file))
	}
	return &types.FileListResponse{List: list, PageResponse: types.PageResponse{PageNo: page, PageSize: size, Total: response.Total}}, nil
}
