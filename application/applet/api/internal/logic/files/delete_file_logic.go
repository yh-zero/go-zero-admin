// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package files

import (
	"context"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除无引用文件，失败可重试
func NewDeleteFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteFileLogic {
	return &DeleteFileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteFileLogic) DeleteFile(req *types.FileIDRequest) (resp *types.MessageResponse, err error) {
	if l.svcCtx.OssClient == nil {
		return nil, xerr.NewErrCodeMsg(300002, "文件存储服务未配置")
	}
	bucket, err := l.svcCtx.OssClient.Bucket(l.svcCtx.Config.Oss.BucketName)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(300002, "文件存储服务配置无效")
	}
	request := &pb.FileIDRequest{Actor: actorSession(l.ctx), ID: req.Id}
	file, err := l.svcCtx.AppletFileRPC.BeginDeleteFile(l.ctx, request)
	if err != nil {
		return nil, err
	}
	if file.Status == "deleted" {
		return &types.MessageResponse{Message: "文件已删除"}, nil
	}
	if err := bucket.DeleteObject(file.ObjectKey, oss.WithContext(l.ctx)); err != nil {
		l.Errorf("OSS file deletion failed id=%d", file.ID)
		return nil, xerr.NewErrCodeMsg(300002, "存储文件删除失败，资源已标记待删除，可再次重试")
	}
	if _, err := l.svcCtx.AppletFileRPC.FinishDeleteFile(l.ctx, request); err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "文件已删除"}, nil
}
