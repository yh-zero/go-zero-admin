package fileservicelogic

import (
	"context"
	"time"

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
	var visibility, objectKey string
	err := l.svcCtx.DB.QueryRowCtx(l.ctx, &visibility,
		"SELECT visibility FROM sys_file WHERE id=? AND deleted_at IS NULL", in.FileID)
	if err == nil && visibility != "" {
		_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &objectKey,
			"SELECT object_key FROM sys_file WHERE id=?", in.FileID)
	}
	if objectKey == "" {
		return nil, xerr.NewErrCodeMsg(404, "文件不存在")
	}
	if l.svcCtx.Store == nil {
		return nil, xerr.NewErrCodeMsg(500, "对象存储未配置")
	}
	ttl := in.TTLSeconds
	if ttl <= 0 || ttl > 3600 {
		ttl = 300
	}
	urlStr, expiresAt, err := l.svcCtx.Store.URL(l.ctx, objectKey, visibility)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(500, "生成链接失败")
	}
	_ = expiresAt
	return &filepb.GetFileURLResponse{URL: urlStr, ExpiresAt: time.Now().Unix() + ttl}, nil
}
