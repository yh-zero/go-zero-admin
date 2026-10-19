package fileresourceservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
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
// 5-minute signed URL for private files. The signer call only happens
// when needed, so callers that already cache a public URL avoid hitting
// OSS.
func (l *GetFileURLLogic) GetFileURL(in *pb.FileIDRequest) (*pb.FileURLResponse, error) {
	if in == nil {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "文件ID无效")
	}
	file, err := scopedFile(l.ctx, l.svcCtx.DB.DB, in.Actor, in.ID, false)
	if err != nil {
		return nil, err
	}
	cfg := l.svcCtx.Config.Oss
	if file.Visibility == "public" {
		return &pb.FileURLResponse{Url: publicFileURL(cfg.BucketName, cfg.Endpoint, file.ObjectKey)}, nil
	}
	bucket, err := ossBucket(l.svcCtx)
	if err != nil {
		return nil, err
	}
	link, err := bucket.SignURL(file.ObjectKey, oss.HTTPGet, privateURLLifetime)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(300002, "文件访问地址生成失败")
	}
	return &pb.FileURLResponse{Url: link, ExpiresIn: privateURLLifetime}, nil
}
