package file

import (
	"context"
	"go-zero-admin/pkg/xerr"

	"go-zero-admin/application/admin/api/internal/middleware"
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
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	viewer, _ := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	if viewer <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	rpcResp, err := l.svcCtx.FileRpc.GetFileList(l.ctx, &filepb.GetFileListRequest{
		Actor:      &filepb.Actor{UserID: viewer},
		Page:       &filepb.PageRequest{Page: req.Page, PageSize: req.PageSize},
		Visibility: req.Visibility, UploaderID: req.UploaderId, IncludeCleanupPending: req.IncludeCleanupPending,
		CleanupPendingOnly: req.CleanupPendingOnly, Name: req.Name, Mime: req.Mime, Ext: req.Ext,
		MinSize: req.MinSize, MaxSize: req.MaxSize, StartTime: req.StartTime, EndTime: req.EndTime,
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
		RefCount: f.RefCount, UploaderId: f.UploaderID, UploaderName: f.UploaderName, SHA256: f.Sha256,
		URL: f.URL, CreatedAt: f.CreatedAt, CleanupPending: f.CleanupPending,
	}
}
