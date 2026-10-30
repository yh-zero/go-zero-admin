package file

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	filepb "go-zero-admin/application/file/rpc/filepb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFileListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetFileListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileListLogic {
	return &GetFileListLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetFileListLogic) GetFileList(req *types.GetFileListReq) (resp *types.GetFileListResp, err error) {
	rpcResp, err := l.svcCtx.FileRpc.GetFileList(l.ctx, &filepb.GetFileListRequest{
		Page:       &filepb.PageRequest{Page: req.Page, PageSize: req.PageSize},
		Visibility: req.Visibility, UploaderID: req.UploaderId,
	})
	if err != nil {
		return nil, err
	}
	list := make([]*types.FileInfo, 0, len(rpcResp.List))
	for _, f := range rpcResp.List {
		list = append(list, mapFileInfo(f))
	}
	return &types.GetFileListResp{List: list, Total: rpcResp.Page.Total}, nil
}

func mapFileInfo(f *filepb.FileInfo) *types.FileInfo {
	return &types.FileInfo{
		Id: f.Id, Bucket: f.Bucket, ObjectKey: f.ObjectKey, OriginalName: f.OriginalName,
		Ext: f.Ext, Mime: f.Mime, Size: f.Size, Visibility: f.Visibility,
		RefCount: f.RefCount, UploaderId: f.UploaderID, URL: f.URL, CreatedAt: f.CreatedAt,
	}
}
