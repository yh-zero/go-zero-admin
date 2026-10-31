package fileservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	filepb "go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/application/file/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetFileURLLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetFileURLLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileURLLogic {
	return &GetFileURLLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetFileURLLogic) GetFileURL(in *filepb.GetFileURLRequest) (*filepb.GetFileURLResponse, error) {
	if !in.PublicOnly && in.GetActor().GetUserID() <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if in.FileID <= 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	row, err := l.svcCtx.SysFileModel.FindLive(l.ctx, in.FileID)
	if err != nil {
		return nil, fileModelError(err)
	}
	if in.PublicOnly && row.Visibility != "public" {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if row.Visibility != "public" && (!row.UploaderId.Valid || row.UploaderId.Int64 != in.Actor.UserID) {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if l.svcCtx.Store == nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	// 期限以存储层真实签名为准：公开文件永久有效返回 0，私有返回实际签名 TTL
	urlStr, expiresAt, err := l.svcCtx.Store.URL(l.ctx, row.ObjectKey, row.Visibility)
	if err != nil {
		l.Error("文件链接生成失败")
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	return &filepb.GetFileURLResponse{URL: urlStr, ExpiresAt: expiresAt}, nil
}
