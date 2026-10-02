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

type GetFileURLLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取受控文件访问地址
func NewGetFileURLLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileURLLogic {
	return &GetFileURLLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetFileURLLogic) GetFileURL(req *types.FileURLRequest) (resp *types.FileURLResponse, err error) {
	file, err := l.svcCtx.AppletFileRPC.GetFile(l.ctx, &pb.FileIDRequest{Actor: actorSession(l.ctx), ID: req.Id})
	if err != nil {
		return nil, err
	}
	if file.Visibility == "public" {
		return &types.FileURLResponse{Url: publicFileURL(l.svcCtx.Config.Oss.BucketName, l.svcCtx.Config.Oss.Endpoint, file.ObjectKey)}, nil
	}
	if l.svcCtx.OssClient == nil {
		return nil, xerr.NewErrCodeMsg(300002, "文件存储服务未配置")
	}
	bucket, err := l.svcCtx.OssClient.Bucket(l.svcCtx.Config.Oss.BucketName)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(300002, "文件存储服务配置无效")
	}
	const lifetime = 300
	link, err := bucket.SignURL(file.ObjectKey, oss.HTTPGet, lifetime)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(300002, "文件访问地址生成失败")
	}
	return &types.FileURLResponse{Url: link, ExpiresIn: lifetime}, nil
}
