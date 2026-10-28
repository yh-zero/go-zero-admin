package fileresourceservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFileURLLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetFileURLLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileURLLogic {
	return &GetFileURLLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// GetFileURL returns the canonical public URL for public files, or a
// signed URL for private files using the configured lifetime.
func (l *GetFileURLLogic) GetFileURL(in *pb.FileIDRequest) (*pb.FileURLResponse, error) {
	if in == nil {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "文件ID无效")
	}
	file, err := scopedFile(l.ctx, l.svcCtx.DB.DB, in.Actor, in.ID, false)
	if err != nil {
		return nil, err
	}
	if file.Status != "active" {
		return nil, fileError("文件已删除或正在删除")
	}
	storage, err := configuredStorage(l.svcCtx)
	if err != nil {
		return nil, err
	}
	link, expires, err := storage.URL(l.ctx, file.ObjectKey, file.Visibility)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(300002, "文件访问地址生成失败")
	}
	return &pb.FileURLResponse{Url: link, ExpiresIn: expires}, nil
}
